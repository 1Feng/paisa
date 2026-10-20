package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
)

// Monetary strings deliberately retain decimal precision independently of the
// legacy API's global decimal JSON marshalling setting.
type CashBalanceRow struct {
	Account         string              `json:"account"`
	Currency        string              `json:"currency"`
	NativeAmount    string              `json:"native_amount"`
	ReferenceAmount *string             `json:"reference_amount"`
	QuoteDate       string              `json:"quote_date,omitempty"`
	Kind            string              `json:"kind"`
	Location        string              `json:"location"`
	Access          string              `json:"access"`
	Reason          string              `json:"reason"`
	LastPosting     string              `json:"last_posting,omitempty"`
	Rule            *config.CashAccount `json:"rule,omitempty"`
}
type CashBalanceGroup struct {
	ReferenceTotal string `json:"reference_total"`
	FXComplete     bool   `json:"fx_complete"`
}
type CashBalance struct {
	AsOf                   string `json:"as_of"`
	Today                  string `json:"today"`
	Currency               string `json:"currency"`
	ReferenceTotal         string `json:"reference_total"`
	FXComplete             bool   `json:"fx_complete"`
	ClassificationComplete bool   `json:"classification_complete"`
	PolicyConfigured       bool   `json:"policy_configured"`
	LastPosting            string `json:"last_posting,omitempty"`
	// Existing caches do not record journal sync timestamps. Do not invent one
	// from the latest transaction or database-file modification time.
	LastJournalSync *string                     `json:"last_journal_sync"`
	Groups          map[string]CashBalanceGroup `json:"groups"`
	Rows            []CashBalanceRow            `json:"rows"`
}

func CashBalanceHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		day := utils.Now().In(config.TimeZone())
		var err error
		if raw, exists := c.GetQuery("date"); exists {
			day, err = time.ParseInLocation("2006-01-02", raw, config.TimeZone())
			if err != nil || day.Format("2006-01-02") > utils.Now().In(config.TimeZone()).Format("2006-01-02") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "date must be a valid YYYY-MM-DD on or before today"})
				return
			}
		}
		var ps []posting.Posting
		var quotes []price.Price
		if err = db.Where("date <= ? AND forecast = ?", utils.EndOfDay(day), false).Find(&ps).Error; err == nil {
			err = db.Where("date <= ? AND commodity_type = ?", utils.EndOfDay(day), config.Unknown).Order("date asc, id asc").Find(&quotes).Error
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cash balance data could not be read"})
			return
		}
		c.JSON(http.StatusOK, computeCashBalance(ps, quotes, config.GetConfig().CashAccounts, config.DefaultCurrency(), day))
	}
}

func computeCashBalance(ps []posting.Posting, quotes []price.Price, rules []config.CashAccount, base string, asOf time.Time) CashBalance {
	base = ledger.NormalizeCurrency(base)
	asOf = asOf.In(config.TimeZone())
	day := asOf.Format("2006-01-02")
	end := utils.EndOfDay(asOf)
	result := CashBalance{AsOf: day, Today: utils.Now().In(config.TimeZone()).Format("2006-01-02"), Currency: base, ReferenceTotal: "0", FXComplete: true, ClassificationComplete: len(rules) > 0, PolicyConfigured: len(rules) > 0, Groups: map[string]CashBalanceGroup{}, Rows: []CashBalanceRow{}}
	type pair struct{ account, currency string }
	quantities := map[pair]decimal.Decimal{}
	last := map[pair]string{}
	known := map[pair]bool{}
	bankAccounts := map[string]bool{}
	brokerAccounts := map[string]bool{}
	active := map[pair]config.CashAccount{}
	for _, r := range rules {
		key := pair{r.Account, r.Currency}
		known[key] = true
		if r.Location == "domestic_bank" || r.Location == "overseas_bank" {
			bankAccounts[r.Account] = true
		}
		if r.Location == "broker" {
			brokerAccounts[r.Account] = true
		}
		if r.Applies(day) {
			active[key] = r
		}
		quantities[key] = decimal.Zero
	}
	for _, p := range ps {
		if p.Forecast || p.Date.After(end) || !utils.IsSameOrParent(p.Account, "Assets") {
			continue
		}
		d := p.Date.In(config.TimeZone()).Format("2006-01-02")
		if d > result.LastPosting {
			result.LastPosting = d
		}
		key := pair{p.Account, ledger.NormalizeCurrency(p.Commodity)}
		_, fiatErr := currency.ParseISO(key.currency)
		// Discovery is broader than inclusion: unknown broker commodities must be
		// reviewed, not guessed to be money or securities. Only an exact active
		// cash rule can add a position to the cash subtotal.
		candidate := known[key] || bankAccounts[p.Account] || brokerAccounts[p.Account] || fiatErr == nil || utils.IsSameOrParent(p.Account, "Assets:Brokerage") || utils.IsSameOrParent(p.Account, "Assets:Broker") || utils.IsSameOrParent(p.Account, "Assets:Bank") || utils.IsSameOrParent(p.Account, "Assets:Wallet") || utils.IsSameOrParent(p.Account, "Assets:Saving") || utils.IsSameOrParent(p.Account, "Assets:Checking")
		if !candidate {
			continue
		}
		quantities[key] = quantities[key].Add(p.Quantity)
		if d > last[key] {
			last[key] = d
		}
	}
	// These prices are journal-native currency quotes already converted to the
	// configured default currency by the historical resolver. Never use posting
	// cost prices or the FX store's backward extrapolation for this balance API.
	latest := map[string]price.Price{}
	for _, q := range quotes {
		if q.Date.After(end) {
			continue
		}
		cur := ledger.NormalizeCurrency(q.CommodityName)
		old, ok := latest[cur]
		if !ok || !q.Date.Before(old.Date) {
			latest[cur] = q
		}
	}
	total := decimal.Zero
	groupSums := map[string]decimal.Decimal{}
	groupComplete := map[string]bool{}
	for key, q := range quantities {
		row := CashBalanceRow{Account: key.account, Currency: key.currency, NativeAmount: q.String(), Kind: "unclassified", Location: "other", Access: "unknown", Reason: "no_active_rule", LastPosting: last[key]}
		if rule, ok := active[key]; ok {
			r := rule
			row.Rule = &r
			row.Kind = r.Kind
			row.Location = r.Location
			row.Access = r.Access
			row.Reason = "explicit_rule"
		}
		if utils.IsSameOrParent(key.account, "Assets:Bridge") || utils.IsSameOrParent(key.account, "Assets:PPA") {
			row.Kind = "excluded"
			row.Reason = "transit_or_retirement"
		}
		if row.Kind == "mixed" || row.Kind == "unclassified" {
			if !q.IsZero() {
				result.ClassificationComplete = false
			}
		}
		if row.Kind == "cash" {
			if _, ok := groupComplete[row.Location]; !ok {
				groupComplete[row.Location] = true
			}
			rate := decimal.NewFromInt(1)
			ok := true
			if key.currency != base {
				quote, exists := latest[key.currency]
				ok = exists && quote.Value.IsPositive() && quote.Available()
				if ok && quote.Conversion != nil {
					ok = !quote.Conversion.QuoteDate.After(end)
					for _, leg := range quote.Conversion.Rates {
						ok = ok && !leg.Date.After(end) && leg.Value.IsPositive()
					}
				}
				if ok {
					rate = quote.Value
					row.QuoteDate = quote.Date.In(config.TimeZone()).Format("2006-01-02")
				}
			}
			// A zero holding has an exactly zero reference value even without FX.
			if ok || q.IsZero() {
				v := q.Mul(rate)
				text := v.String()
				row.ReferenceAmount = &text
				total = total.Add(v)
				groupSums[row.Location] = groupSums[row.Location].Add(v)
			} else {
				result.FXComplete = false
				groupComplete[row.Location] = false
			}
		}
		result.Rows = append(result.Rows, row)
	}
	for location, complete := range groupComplete {
		result.Groups[location] = CashBalanceGroup{ReferenceTotal: groupSums[location].String(), FXComplete: complete}
	}
	result.ReferenceTotal = total.String()
	sort.Slice(result.Rows, func(i, j int) bool {
		a, b := result.Rows[i], result.Rows[j]
		if a.Account != b.Account {
			return a.Account < b.Account
		}
		return strings.Compare(a.Currency, b.Currency) < 0
	})
	return result
}
