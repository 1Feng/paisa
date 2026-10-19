package service

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMarketExplicitZeroIsNotMissingPrice(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{DefaultCurrency: "CNY"})
	t.Cleanup(func() { config.SetConfigForTest(prev); ClearPriceCache() })
	ClearPriceCache()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&price.Price{}, &posting.Posting{}))
	d := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	p := posting.Posting{Account: "Assets:Broker", Commodity: "ZERO", Date: d, Quantity: decimal.NewFromInt(10), Amount: decimal.NewFromInt(100)}
	require.NoError(t, db.Create(&p).Error)
	require.NoError(t, db.Create(&price.Price{Date: d, CommodityName: "ZERO", CommodityType: config.Unknown, Value: decimal.Zero}).Error)
	require.True(t, GetMarketPrice(db, p, d).IsZero(), "an explicit zero quote must not fall back to cost")
	require.True(t, GetPrice(db, "ZERO", decimal.NewFromInt(10), d).IsZero())
}
