package price

import (
	"time"

	"gorm.io/gorm"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/google/btree"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type Price struct {
	ID            uint                 `gorm:"primaryKey" json:"id"`
	Date          time.Time            `json:"date"`
	CommodityType config.CommodityType `json:"commodity_type"`
	CommodityID   string               `json:"commodity_id"`
	CommodityName string               `json:"commodity_name"`
	Value         decimal.Decimal      `json:"value"`
	Conversion    *Conversion          `gorm:"serializer:json" json:"conversion,omitempty"`
}

// Conversion describes a journal quote expressed in a non-default currency.
// Date remains the effective valuation date; QuoteDate is the native quote's
// date, so an FX-only update never masquerades as a fresh security quote.
type Conversion struct {
	Status        string           `json:"status"`
	QuoteCurrency string           `json:"quote_currency"`
	QuoteValue    decimal.Decimal  `json:"quote_value"`
	QuoteDate     time.Time        `json:"quote_date"`
	Rates         []ConversionRate `json:"rates,omitempty"`
}

type ConversionRate struct {
	From  string          `json:"from"`
	To    string          `json:"to"`
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}

func (p Price) Less(o btree.Item) bool {
	return p.Date.Before(o.(Price).Date)
}

// Available distinguishes an explicit zero quote from an absent FX conversion.
func (p Price) Available() bool {
	return !p.Date.IsZero() && (p.Conversion == nil || p.Conversion.Status != "missing_fx")
}

func DeleteAll(db *gorm.DB) error {
	err := db.Exec("DELETE FROM prices").Error
	if err != nil {
		return err
	}
	return nil
}

func UpsertAllByTypeNameAndID(db *gorm.DB, commodityType config.CommodityType, commodityName string, commodityID string, prices []*Price) {
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Delete(&Price{}, "commodity_type = ? and (commodity_id = ? or commodity_name = ?)", commodityType, commodityID, commodityName).Error
		if err != nil {
			return err
		}

		for _, price := range prices {
			err := tx.Create(price).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func UpsertAllByType(db *gorm.DB, commodityType config.CommodityType, prices []Price) {
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Delete(&Price{}, "commodity_type = ?", commodityType).Error
		if err != nil {
			return err
		}
		for _, price := range prices {
			err := tx.Create(&price).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
