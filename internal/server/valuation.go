package server

import (
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ValuationGap struct {
	Commodity string     `json:"commodity"`
	Date      time.Time  `json:"date"`
	Until     *time.Time `json:"until_date,omitempty"`
	Currency  string     `json:"currency"`
}

// GetValuationCoverage is additive: existing monetary API fields retain their
// compatibility estimates, while clients can identify incomplete history.
func GetValuationCoverage(db *gorm.DB) gin.H {
	var ps []posting.Posting
	db.Where("date <= ?", utils.EndOfToday()).Find(&ps)
	names := map[string]bool{}
	holdings := map[string]decimal.Decimal{}
	for _, p := range ps {
		if ledger.NormalizeCurrency(p.Commodity) == ledger.NormalizeCurrency(config.DefaultCurrency()) {
			continue
		}
		names[p.Commodity] = true
		if utils.IsSameOrParent(p.Account, "Assets") {
			holdings[p.Commodity] = holdings[p.Commodity].Add(p.Quantity)
		}
	}
	gaps := []ValuationGap{}
	missing := []string{}
	hasConversions := false
	for name := range names {
		prices := service.GetAllPrices(db, name)
		for i, p := range prices {
			if p.Conversion == nil {
				continue
			}
			hasConversions = true
			if p.Conversion.Status != "missing_fx" || p.Date.After(utils.EndOfToday()) {
				continue
			}
			resolved := service.GetUnitPrice(db, name, p.Date)
			if resolved.Conversion == nil || resolved.Conversion.Status != "missing_fx" {
				continue
			}
			gap := ValuationGap{Commodity: name, Date: p.Date, Currency: p.Conversion.QuoteCurrency}
			// GetAllPrices is descending. The immediately newer record is
			// the end of this point's applicability, even if also missing.
			if i > 0 {
				end := prices[i-1].Date
				gap.Until = &end
			}
			gaps = append(gaps, gap)
		}
		if !holdings[name].IsZero() {
			p := service.GetUnitPrice(db, name, utils.EndOfToday())
			if p.Date.IsZero() || p.Conversion != nil && p.Conversion.Status == "missing_fx" {
				missing = append(missing, name)
			}
		}
	}
	sort.Strings(missing)
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Commodity != gaps[j].Commodity {
			return gaps[i].Commodity < gaps[j].Commodity
		}
		return gaps[i].Date.Before(gaps[j].Date)
	})
	return gin.H{"gaps": gaps, "current_missing": missing, "current_complete": len(missing) == 0, "has_conversions": hasConversions}
}
