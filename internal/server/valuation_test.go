package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestValuationCoveragePersistsHistoricalGapAndCurrentMissing(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{DefaultCurrency: "CNY"})
	now := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	utils.SetNow("2024-01-03")
	t.Cleanup(func() { config.SetConfigForTest(prev); utils.ResetNow(); service.ClearPriceCache() })
	service.ClearPriceCache()
	db := inMemoryDB(t)
	for _, s := range []string{"TEST", "NOQUOTE", "¥"} {
		require.NoError(t, db.Create(&posting.Posting{Date: now.AddDate(0, 0, -2), Account: "Assets:Broker", Commodity: s, Quantity: decimal.NewFromInt(1)}).Error)
	}
	rows := []price.Price{{Date: now.AddDate(0, 0, -2), CommodityName: "TEST", CommodityType: config.Unknown, Conversion: &price.Conversion{Status: "missing_fx", QuoteCurrency: "USD", QuoteValue: decimal.NewFromInt(100), QuoteDate: now.AddDate(0, 0, -2)}}, {Date: now.AddDate(0, 0, -1), CommodityName: "TEST", CommodityType: config.Unknown, Value: decimal.NewFromInt(700)}}
	require.NoError(t, db.Create(&rows).Error)
	for i := 0; i < 2; i++ {
		service.ClearPriceCache() // Simulate rebuilding caches after restart.
		got := GetValuationCoverage(db)
		require.False(t, got["current_complete"].(bool))
		require.Equal(t, []string{"NOQUOTE"}, got["current_missing"])
		gaps := got["gaps"].([]ValuationGap)
		require.Len(t, gaps, 1)
		require.Equal(t, "TEST", gaps[0].Commodity)
		require.Equal(t, rows[1].Date, gaps[0].Until.UTC())
	}
}

func TestNetworthTimelineHonorsExplicitZeroQuote(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{DefaultCurrency: "CNY"})
	utils.SetNow("2024-01-03")
	t.Cleanup(func() {
		config.SetConfigForTest(prev)
		utils.ResetNow()
		service.ClearPriceCache()
		service.ClearInterestCache()
	})
	service.ClearPriceCache()
	service.ClearInterestCache()
	db := inMemoryDB(t)
	d := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	p := posting.Posting{Account: "Assets:Broker", Commodity: "ZERO", Date: d, Quantity: decimal.NewFromInt(10), Amount: decimal.NewFromInt(100)}
	require.NoError(t, db.Create(&p).Error)
	require.NoError(t, db.Create(&[]price.Price{{Date: d, CommodityName: "ZERO", CommodityType: config.Unknown, Value: decimal.NewFromInt(10)}, {Date: d.AddDate(0, 0, 1), CommodityName: "ZERO", CommodityType: config.Unknown}}).Error)
	points := computeNetworthTimeline(db, []posting.Posting{p}, false, nil)
	require.True(t, points[len(points)-1].BalanceAmount.IsZero(), "zero quote must not become the acquisition cost in the timeline")
}
