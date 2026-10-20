package model

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

func TestJournalReplacementRollbackAndConversionRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&price.Price{}, &posting.Posting{}))
	d := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	old := price.Price{Date: d, CommodityName: "OLD", CommodityType: config.Unknown, Value: decimal.NewFromInt(5)}
	require.NoError(t, db.Create(&old).Error)
	require.NoError(t, db.Create(&posting.Posting{Account: "Assets:Old"}).Error)
	require.NoError(t, db.Exec("CREATE TRIGGER reject_posting BEFORE INSERT ON postings BEGIN SELECT RAISE(ABORT, 'rejected'); END").Error)
	p := price.Price{Date: d, CommodityName: "NEW", CommodityType: config.Unknown, Value: decimal.NewFromInt(700), Conversion: &price.Conversion{Status: "converted", QuoteCurrency: "USD", QuoteValue: decimal.NewFromInt(100), QuoteDate: d, Rates: []price.ConversionRate{{From: "USD", To: "CNY", Date: d, Value: decimal.NewFromInt(7)}}}}
	require.Error(t, replaceJournal(db, []price.Price{p}, []*posting.Posting{{Account: "Assets:New"}}))
	var rows []price.Price
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "OLD", rows[0].CommodityName)
	var ps []posting.Posting
	require.NoError(t, db.Find(&ps).Error)
	require.Equal(t, "Assets:Old", ps[0].Account)
	require.NoError(t, db.Exec("DROP TRIGGER reject_posting").Error)
	require.NoError(t, replaceJournal(db, []price.Price{p}, nil))
	rows = nil
	require.NoError(t, db.Find(&rows).Error)
	require.Equal(t, "USD", rows[0].Conversion.QuoteCurrency)
	require.Equal(t, "7", rows[0].Conversion.Rates[0].Value.String())
}
