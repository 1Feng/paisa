package ledger

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type nativeQuote struct {
	date           time.Time
	name, currency string
	value          decimal.Decimal
}

type journalRateBook map[string][]price.ConversionRate

// Unlike fx.RateStore, this resolver deliberately never extrapolates backwards.
func (book journalRateBook) lookup(from, to string, day time.Time) (decimal.Decimal, []price.ConversionRate, bool) {
	if from == to {
		return decimal.NewFromInt(1), nil, true
	}
	lookup := func(a, b string) (price.ConversionRate, bool) {
		points := book[a+":"+b]
		i := sort.Search(len(points), func(i int) bool { return points[i].Date.After(utils.EndOfDay(day)) })
		if i == 0 || !points[i-1].Value.IsPositive() {
			return price.ConversionRate{}, false
		}
		return points[i-1], true
	}
	if r, ok := lookup(from, to); ok {
		return r.Value, []price.ConversionRate{r}, true
	}
	if r, ok := lookup(to, from); ok {
		r.From, r.To = from, to
		r.Value = decimal.NewFromInt(1).Div(r.Value)
		return r.Value, []price.ConversionRate{r}, true
	}
	if from != "USD" && to != "USD" {
		left, ls, lok := book.lookup(from, "USD", day)
		right, rs, rok := book.lookup("USD", to, day)
		if lok && rok {
			return left.Mul(right), append(ls, rs...), true
		}
	}
	return decimal.Zero, nil, false
}

func convertJournalPrices(output, defaultCurrency string) ([]price.Price, error) {
	base := NormalizeCurrency(defaultCurrency)
	byName := map[string][]nativeQuote{}
	// Currency nodes must occur as denominations or be established symbol
	// aliases. An ISO-looking ticker (e.g. AMD) alone is not an FX edge.
	currencies := map[string]bool{base: true}
	for _, iso := range symbolToISO {
		currencies[iso] = true
	}
	var rows []nativeQuote
	re := regexp.MustCompile(`^P (\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) ("[^"]+"|\S+) (.+)$`)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := re.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			return nil, fmt.Errorf("invalid journal price: %s", line)
		}
		date, err := time.ParseInLocation("2006/01/02 15:04:05", m[1], config.TimeZone())
		if err != nil {
			return nil, err
		}
		cur, value, err := parseAmount(m[3])
		if err != nil {
			return nil, err
		}
		cur = NormalizeCurrency(cur)
		if _, err := currency.ParseISO(cur); err == nil {
			currencies[cur] = true
		}
		q := nativeQuote{date: date, name: utils.UnQuote(m[2]), currency: cur, value: value}
		rows = append(rows, q)
		byName[q.name] = append(byName[q.name], q)
	}
	book := journalRateBook{}
	events := map[time.Time]bool{}
	for _, q := range rows {
		from := NormalizeCurrency(q.name)
		if !currencies[from] || !currencies[q.currency] || from == q.currency {
			continue
		}
		book[from+":"+q.currency] = append(book[from+":"+q.currency], price.ConversionRate{From: from, To: q.currency, Date: q.date, Value: q.value})
		events[journalDay(q.date)] = true
	}
	for key, points := range book {
		sort.Slice(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })
		for i := 1; i < len(points); i++ {
			if points[i].Date.Equal(points[i-1].Date) && !points[i].Value.Equal(points[i-1].Value) {
				return nil, fmt.Errorf("conflicting FX quotes for %s at %s", key, points[i].Date)
			}
		}
		book[key] = points
	}
	makePrice := func(q nativeQuote, day time.Time) price.Price {
		return price.Price{Date: day, CommodityName: q.name, CommodityID: q.name, CommodityType: config.Unknown, Value: q.value}
	}
	var out []price.Price
	// Keep each legacy day's first slot (and fixture IDs), but choose its
	// latest native timestamp independently of input order.
	type directKey struct {
		name string
		day  time.Time
	}
	directSlots := map[directKey]int{}
	directTimes := map[directKey]time.Time{}
	for _, q := range rows {
		if q.currency == base {
			key := directKey{q.name, journalDay(q.date)}
			p := makePrice(q, key.day)
			if currencies[NormalizeCurrency(q.name)] && !q.value.IsPositive() && NormalizeCurrency(q.name) != base {
				// A currency rate must be positive. Security zero quotes remain
				// valid, but invalid FX must also be unavailable for cash holdings.
				p.Conversion = &price.Conversion{Status: "missing_fx", QuoteCurrency: q.currency, QuoteValue: q.value, QuoteDate: q.date}
			}
			if slot, exists := directSlots[key]; exists {
				if q.date.After(directTimes[key]) {
					out[slot] = p
					directTimes[key] = q.date
				}
			} else {
				directSlots[key] = len(out)
				directTimes[key] = q.date
				out = append(out, p)
			}
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		qs := byName[name]
		sort.Slice(qs, func(i, j int) bool {
			if !qs[i].date.Equal(qs[j].date) {
				return qs[i].date.Before(qs[j].date)
			}
			if (qs[i].currency == base) != (qs[j].currency == base) {
				return qs[j].currency == base
			}
			return qs[i].currency < qs[j].currency
		})
		for i := 1; i < len(qs); i++ {
			if qs[i].date.Equal(qs[i-1].date) && qs[i].currency == qs[i-1].currency && !qs[i].value.Equal(qs[i-1].value) {
				return nil, fmt.Errorf("conflicting journal quotes for %s at %s", name, qs[i].date)
			}
		}
		days := map[time.Time]bool{}
		for d := range events {
			days[d] = true
		}
		for _, q := range qs {
			days[journalDay(q.date)] = true
		}
		ordered := make([]time.Time, 0, len(days))
		for d := range days {
			ordered = append(ordered, d)
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Before(ordered[j]) })
		index := 0
		var previous *price.Price
		for _, day := range ordered {
			for index < len(qs) && !qs[index].date.After(utils.EndOfDay(day)) {
				index++
			}
			if index == 0 {
				continue
			}
			q := qs[index-1]
			if q.currency == base {
				previous = nil
				continue
			}
			// A reverse quotation of the base currency is an FX edge, not
			// a security whose value needs to be shown in base currency.
			if NormalizeCurrency(name) == base {
				continue
			}
			p := makePrice(q, day)
			rate, legs, ok := book.lookup(q.currency, base, day)
			p.Conversion = &price.Conversion{Status: "converted", QuoteCurrency: q.currency, QuoteValue: q.value, QuoteDate: q.date, Rates: legs}
			if ok {
				p.Value = q.value.Mul(rate)
			} else {
				p.Value = decimal.Zero
				p.Conversion.Status = "missing_fx"
			}
			if previous != nil && sameConversion(*previous, p) {
				continue
			}
			out = append(out, p)
			previous = &out[len(out)-1]
		}
	}
	return out, nil
}

func sameConversion(a, b price.Price) bool {
	x, y := a.Conversion, b.Conversion
	if x.Status != y.Status || !x.QuoteDate.Equal(y.QuoteDate) || !x.QuoteValue.Equal(y.QuoteValue) || x.QuoteCurrency != y.QuoteCurrency || len(x.Rates) != len(y.Rates) || !a.Value.Equal(b.Value) {
		return false
	}
	for i, r := range x.Rates {
		s := y.Rates[i]
		if r.From != s.From || r.To != s.To || !r.Date.Equal(s.Date) || !r.Value.Equal(s.Value) {
			return false
		}
	}
	return true
}

func journalDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
