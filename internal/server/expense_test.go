package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/account"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// makeExpensePosting builds a minimal expense posting fixture. In paisa's
// convention an Expenses:* posting is positive when it is a real expense
// and negative when it is a refund (红冲/退款) that cancels part of a prior
// expense. This is the M0-C / M3-G refund discipline.
func makeExpensePosting(account string, amount float64, date time.Time) posting.Posting {
	return posting.Posting{
		Account:      account,
		Amount:       decimal.NewFromFloat(amount),
		MarketAmount: decimal.NewFromFloat(amount),
		Quantity:     decimal.NewFromFloat(amount),
		Commodity:    "CNY",
		Date:         date,
	}
}

// TestComputeExpenseSummary_RefundReducesNet locks in the issue #25
// invariant: when a month has a +30000 Expenses:Train purchase and a
// -2990 Expenses:Train refund (style A), the per-month summary must
// expose gross=30000, refunds=-2990, net=27010 so the UI toggle has
// real numbers to switch between.
func TestComputeExpenseSummary_RefundReducesNet(t *testing.T) {
	d := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	postings := []posting.Posting{
		makeExpensePosting("Expenses:Transport:Train", 30000, d),
		makeExpensePosting("Expenses:Transport:Train", -2990, d.AddDate(0, 0, 3)),
	}

	summaries := computeExpenseSummary(postings, "2006-01")

	got, ok := summaries["2024-06"]
	assert.True(t, ok, "expected 2024-06 summary, keys=%v", summaries)
	assert.True(t, got.Gross.Equal(decimal.NewFromInt(30000)),
		"gross: got %s want 30000", got.Gross.String())
	assert.True(t, got.Refunds.Equal(decimal.NewFromInt(-2990)),
		"refunds: got %s want -2990", got.Refunds.String())
	assert.True(t, got.Net.Equal(decimal.NewFromInt(27010)),
		"net: got %s want 27010", got.Net.String())
}

// TestComputeExpenseSummary_NoRefund — a month with only forward
// expenses has refunds=0 and gross == net. This is the common path
// and must not regress.
func TestComputeExpenseSummary_NoRefund(t *testing.T) {
	d := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	postings := []posting.Posting{
		makeExpensePosting("Expenses:Food", 100, d),
		makeExpensePosting("Expenses:Food", 250, d.AddDate(0, 0, 5)),
	}

	summaries := computeExpenseSummary(postings, "2006-01")

	got := summaries["2024-07"]
	assert.True(t, got.Gross.Equal(decimal.NewFromInt(350)))
	assert.True(t, got.Refunds.IsZero(), "no refund → refunds must be zero, got %s", got.Refunds.String())
	assert.True(t, got.Net.Equal(decimal.NewFromInt(350)))
}

// expenseInMemoryDB returns an in-memory sqlite gorm.DB with only the tables
// computeExpenseInvestments touches already migrated.
func expenseInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&posting.Posting{}))
	assert.NoError(t, db.AutoMigrate(&price.Price{}))
	return db
}

func makeAssetLeg(txID, account, commodity string, amount float64, date time.Time) posting.Posting {
	return posting.Posting{
		TransactionID: txID,
		Date:          date,
		Account:       account,
		Commodity:     commodity,
		Quantity:      decimal.NewFromFloat(amount),
		Amount:        decimal.NewFromFloat(amount),
		MarketAmount:  decimal.NewFromFloat(amount),
	}
}

// TestComputeExpenseInvestments_DropsSpendingFromSavings — issue #64 R5:
// the /expense/monthly page treated every Assets:* posting (minus the
// hard-coded Assets:Checking) as an investment. A month of regular
// spending therefore produced a large NEGATIVE "净投资" (-¥23,026.51 /
// -639% of net income on mydata). The fix routes the investments field
// through the same M1-E kind filter the /api/investment page uses, so
// bank_current / bank_savings legs of an Expenses:* transaction drop
// out and the tile reflects real investment flow.
func TestComputeExpenseInvestments_DropsSpendingFromSavings(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{
		DefaultCurrency: "CNY",
		BaseCurrency:    "CNY",
		TimeZone:        "UTC",
		Locale:          "zh-CN",
		Accounts: []config.Account{
			{Name: "Assets:Saving:CMB", Kind: account.BankCurrent},
		},
	})
	defer config.SetConfigForTest(prev)
	service.ClearInterestCache()
	service.ClearPriceCache()
	transaction.ClearCache()

	db := expenseInMemoryDB(t)
	d := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	db.Create(&[]posting.Posting{
		makeAssetLeg("tx-spend", "Assets:Saving:CMB", "CNY", -85, d),
	})

	got := computeExpenseInvestments(db)
	for _, p := range got {
		assert.NotEqual(t, "Assets:Saving:CMB", p.Account,
			"a bank_current spending leg leaked into net investment (R5 regression)")
	}
	assert.Empty(t, got, "month with only spending should report 0 investment postings")
}

// TestComputeExpenseInvestments_KeepsRealBrokeragePurchase — sanity check
// that a genuine brokerage buy (mutual_fund kind) IS counted.
func TestComputeExpenseInvestments_KeepsRealBrokeragePurchase(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{
		DefaultCurrency: "CNY",
		BaseCurrency:    "CNY",
		TimeZone:        "UTC",
		Locale:          "zh-CN",
		Accounts: []config.Account{
			{Name: "Assets:Brokerage:DanJuan", Kind: account.MutualFund},
			{Name: "Assets:Saving:CMB", Kind: account.BankCurrent},
		},
	})
	defer config.SetConfigForTest(prev)
	service.ClearInterestCache()
	service.ClearPriceCache()
	transaction.ClearCache()

	db := expenseInMemoryDB(t)
	d := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	db.Create(&[]posting.Posting{
		makeAssetLeg("tx-buy", "Assets:Brokerage:DanJuan", "CNY", 1000, d),
		makeAssetLeg("tx-buy", "Assets:Saving:CMB", "CNY", -1000, d),
	})

	got := computeExpenseInvestments(db)
	assert.Len(t, got, 1, "exactly one investment leg should survive")
	assert.Equal(t, "Assets:Brokerage:DanJuan", got[0].Account)
	assert.True(t, got[0].Amount.Equal(decimal.NewFromInt(1000)),
		"brokerage buy amount should be +1000, got %s", got[0].Amount.String())
}

// TestComputeExpenseInvestments_DropsBridgeTransfers — M0-B
// `transfer_accounts` integration. A Bridge transfer is not a fresh
// investment; the net effect on the portfolio is zero.
func TestComputeExpenseInvestments_DropsBridgeTransfers(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{
		DefaultCurrency:  "CNY",
		BaseCurrency:     "CNY",
		TimeZone:         "UTC",
		Locale:           "zh-CN",
		TransferAccounts: []string{"Assets:Bridge:*"},
		Accounts: []config.Account{
			{Name: "Assets:Brokerage:PASC", Kind: account.Stock},
			{Name: "Assets:Bridge:Brokerage:PASC", Kind: account.CashEquivalent},
		},
	})
	defer config.SetConfigForTest(prev)
	service.ClearInterestCache()
	service.ClearPriceCache()
	transaction.ClearCache()

	db := expenseInMemoryDB(t)
	d := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	db.Create(&[]posting.Posting{
		makeAssetLeg("tx-bridge", "Assets:Brokerage:PASC", "CNY", 2500, d),
		makeAssetLeg("tx-bridge", "Assets:Bridge:Brokerage:PASC", "CNY", -2500, d),
	})

	got := computeExpenseInvestments(db)
	assert.Empty(t, got, "Bridge transfer should be fully filtered out, got %d postings", len(got))
}

// TestComputeExpenseInvestments_MydataMayFixture replays the exact bug
// scenario from issue #64 R5 (May 2026 on mydata): one cross-currency
// trans, one savings deposit, and several regular expense legs. None of
// these should count as "net investment". Before the fix this returned
// -¥23,026.76 (= -639% of net income).
func TestComputeExpenseInvestments_MydataMayFixture(t *testing.T) {
	prev := config.SetConfigForTest(config.Config{
		DefaultCurrency:  "CNY",
		BaseCurrency:     "CNY",
		TimeZone:         "UTC",
		Locale:           "zh-CN",
		TransferAccounts: []string{"Assets:Bridge:*"},
		Accounts: []config.Account{
			{Name: "Assets:Saving:CMB", Kind: account.BankCurrent},
			{Name: "Assets:Saving:HSBC-CN", Kind: account.BankCurrent},
			{Name: "Assets:Saving:HSBC-US", Kind: account.BankCurrent},
			{Name: "Assets:Saving:CMBC-HK", Kind: account.BankCurrent},
			{Name: "Assets:Wallet:Wasabicard:7260", Kind: account.CashEquivalent},
			{Name: "Assets:Wallet:Wasabicard:Wallet", Kind: account.CashEquivalent},
		},
	})
	defer config.SetConfigForTest(prev)
	service.ClearInterestCache()
	service.ClearPriceCache()
	transaction.ClearCache()

	db := expenseInMemoryDB(t)
	d := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	db.Create(&[]posting.Posting{
		makeAssetLeg("tx-trans-cn", "Assets:Saving:HSBC-CN", "CNY", 20000, d),
		makeAssetLeg("tx-trans-cn", "Assets:Saving:CMBC-HK", "CNY", -20000, d),
		makeAssetLeg("tx-reimb", "Assets:Saving:CMB", "CNY", 3381.15, d.AddDate(0, 0, 18)),
		makeAssetLeg("tx-wasabi", "Assets:Wallet:Wasabicard:7260", "CNY", -697.30, d.AddDate(0, 0, 25)),
		makeAssetLeg("tx-cmb-spend", "Assets:Saving:CMB", "CNY", -25710.61, d.AddDate(0, 0, 20)),
	})

	got := computeExpenseInvestments(db)
	sum := decimal.Zero
	for _, p := range got {
		sum = sum.Add(p.Amount)
	}
	assert.True(t, sum.IsZero(),
		"R5: mydata May 2026 net investment must be 0 (saving + wallet flows only), got %s",
		sum.String())
	assert.Empty(t, got,
		"R5: every leg in mydata's May 2026 is bank/cash kind, expected zero survivors, got %d",
		len(got))
}

// TestComputeExpenseSummary_YearlyKey verifies the helper honors the
// supplied date layout so it works for both month_wise (2006-01) and
// year_wise (2006) aggregation.
func TestComputeExpenseSummary_YearlyKey(t *testing.T) {
	postings := []posting.Posting{
		makeExpensePosting("Expenses:Travel", 5000, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)),
		makeExpensePosting("Expenses:Travel", -500, time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)),
		makeExpensePosting("Expenses:Travel", 1000, time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)),
	}

	summaries := computeExpenseSummary(postings, "2006")

	y2024 := summaries["2024"]
	assert.True(t, y2024.Gross.Equal(decimal.NewFromInt(5000)))
	assert.True(t, y2024.Refunds.Equal(decimal.NewFromInt(-500)))
	assert.True(t, y2024.Net.Equal(decimal.NewFromInt(4500)))

	y2023 := summaries["2023"]
	assert.True(t, y2023.Gross.Equal(decimal.NewFromInt(1000)))
	assert.True(t, y2023.Refunds.IsZero())
	assert.True(t, y2023.Net.Equal(decimal.NewFromInt(1000)))
}
