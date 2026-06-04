package ledger

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/google/btree"
	"github.com/shopspring/decimal"
)

// makeHLedgerPosting builds an HLedgerPosting with a single amount whose
// commodity / price commodity are supplied as (possibly symbol) currencies.
func makeHLedgerPosting(commodity string, qty float64, priceCommodity string, priceQty float64, priceTag string) HLedgerPosting {
	p := HLedgerPosting{Account: "Assets:Cash"}
	a := struct {
		Commodity string `json:"acommodity"`
		Quantity  struct {
			Value float64 `json:"floatingPoint"`
		} `json:"aquantity"`
		Price struct {
			Contents struct {
				Commodity string `json:"acommodity"`
				Quantity  struct {
					Value float64 `json:"floatingPoint"`
				} `json:"aquantity"`
			} `json:"contents"`
			Tag string `json:"tag"`
		} `json:"aprice"`
	}{}
	a.Commodity = commodity
	a.Quantity.Value = qty
	a.Price.Contents.Commodity = priceCommodity
	a.Price.Contents.Quantity.Value = priceQty
	a.Price.Tag = priceTag
	p.Amount = append(p.Amount, a)
	return p
}

func makeHLedgerTransaction() HLedgerTransaction {
	t := HLedgerTransaction{ID: 1, Description: "Test", Status: "*"}
	t.TSourcePos = []struct {
		SourceColumn uint64 `json:"sourceColumn"`
		SourceLine   uint64 `json:"sourceLine"`
		SourceName   string `json:"sourceName"`
	}{
		{SourceLine: 1, SourceName: "main.ledger"},
		{SourceLine: 2, SourceName: "main.ledger"},
	}
	return t
}

// TestBuildHLedgerPostingsSymbolCommodityIsDefault verifies that when a posting's
// commodity is the SYMBOL form of the default currency (e.g. ¥ for default CNY),
// it is treated as the default currency: Amount should equal the raw quantity and
// no price conversion should occur.
func TestBuildHLedgerPostingsSymbolCommodityIsDefault(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{DefaultCurrency: "CNY"})
	t.Cleanup(func() { config.SetConfigForTest(prev) })

	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Seed a price for "¥" so the bug is observable: if the symbol is NOT
	// normalized, the code wrongly treats "¥" as a non-default commodity and
	// looks it up in the price tree, multiplying 100 * 7 = 700. When normalized,
	// "¥" == default currency (CNY) and no conversion happens (Amount stays 100).
	pricesTree := map[string]*btree.BTree{
		"¥": btree.New(2),
	}
	pricesTree["¥"].ReplaceOrInsert(price.Price{Date: date, Value: decimal.NewFromInt(7)})

	p := makeHLedgerPosting("¥", 100, "", 0, "")
	tx := makeHLedgerTransaction()
	tx.Postings = []HLedgerPosting{p}

	postings, err := buildHLedgerPostings(p, tx, pricesTree, date)
	if err != nil {
		t.Fatalf("buildHLedgerPostings returned error: %v", err)
	}
	if len(postings) != 1 {
		t.Fatalf("expected 1 posting, got %d", len(postings))
	}

	// ¥ normalizes to CNY (the default currency), so Amount must equal the raw
	// quantity 100 (no price lookup / conversion).
	if !postings[0].Amount.Equal(postings[0].Quantity) {
		t.Errorf("expected Amount (%s) to equal Quantity (%s) for default currency posting",
			postings[0].Amount, postings[0].Quantity)
	}
}

// TestBuildHLedgerPostingsSymbolPriceCommodityIsDefault drives the inline-price
// branch: a non-default commodity priced in the SYMBOL form of the default
// currency. The unconverted total (qty * price) should be used directly without
// any price-tree lookup, because the price commodity normalizes to the default.
func TestBuildHLedgerPostingsSymbolPriceCommodityIsDefault(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{DefaultCurrency: "CNY"})
	t.Cleanup(func() { config.SetConfigForTest(prev) })

	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Empty price tree: if the code wrongly takes the "price commodity is not
	// default" branch it will look up AAPL / ¥ in the tree, find nothing, and
	// leave Amount at the raw quantity (10) instead of the unconverted total.
	pricesTree := map[string]*btree.BTree{}

	// 10 AAPL @ ¥150 each -> unconverted total 1500, in default currency (CNY).
	p := makeHLedgerPosting("AAPL", 10, "¥", 150, "UnitPrice")
	tx := makeHLedgerTransaction()
	tx.Postings = []HLedgerPosting{p}

	postings, err := buildHLedgerPostings(p, tx, pricesTree, date)
	if err != nil {
		t.Fatalf("buildHLedgerPostings returned error: %v", err)
	}
	if len(postings) != 1 {
		t.Fatalf("expected 1 posting, got %d", len(postings))
	}

	want := "1500"
	if postings[0].Amount.String() != want {
		t.Errorf("expected Amount %s (unconverted total, price commodity normalizes to default), got %s",
			want, postings[0].Amount)
	}
}
