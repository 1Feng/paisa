package server

import (
	"encoding/json"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func cashRule(account, cur, kind string) config.CashAccount {
	return config.CashAccount{Account: account, Currency: cur, Kind: kind, Location: "domestic_bank", Access: "unknown", EffectiveFrom: "2024-01-02", Evidence: "Synthetic confirmation"}
}
func cashPosting(a, c, q, day string) posting.Posting {
	return posting.Posting{Account: a, Commodity: c, Quantity: decimal.RequireFromString(q), Amount: decimal.NewFromInt(99999), Date: cashDay(day)}
}
func cashDay(day string) time.Time {
	d, _ := time.ParseInLocation("2006-01-02", day, config.TimeZone())
	return d
}
func TestCashBalanceExactPairsDatesAndSigns(t *testing.T) {
	ps := []posting.Posting{
		cashPosting("Assets:Bank", "¥", "100", "2024-01-01"), cashPosting("Assets:Bank:Fixed", "¥", "500", "2024-01-01"),
		cashPosting("Assets:Broker", "$", "10", "2024-01-01"), cashPosting("Assets:Broker", "GOOG", "2", "2024-01-01"),
		cashPosting("Assets:Other", "CNY", "-20", "2024-01-01"), cashPosting("Assets:Mixed", "CNY", "1000", "2024-01-01"),
		cashPosting("Assets:New", "CNY", "1", "2024-01-01"), cashPosting("Assets:Wallet", "USDT", "5", "2024-01-01"),
		cashPosting("Assets:Bank", "CNY", "999", "2024-01-03"),
	}
	f := cashPosting("Assets:Bank", "CNY", "999", "2024-01-02")
	f.Forecast = true
	ps = append(ps, f)
	rules := []config.CashAccount{cashRule("Assets:Bank", "CNY", "cash"), cashRule("Assets:Bank:Fixed", "CNY", "excluded"), cashRule("Assets:Broker", "USD", "cash"), cashRule("Assets:Other", "CNY", "cash"), cashRule("Assets:Mixed", "CNY", "mixed"), cashRule("Assets:Zero", "CNY", "cash")}
	rules[2].Location = "broker" // Broker securities must not be treated as unfamiliar bank money.
	quotes := []price.Price{{CommodityName: "USD", Date: cashDay("2024-01-01"), Value: decimal.NewFromInt(7)}, {CommodityName: "USD", Date: cashDay("2024-01-03"), Value: decimal.NewFromInt(99)}}
	got := computeCashBalance(ps, quotes, rules, "CNY", cashDay("2024-01-02"))
	require.Equal(t, "150", got.ReferenceTotal)
	require.True(t, got.FXComplete)
	require.False(t, got.ClassificationComplete)
	rows := map[string]CashBalanceRow{}
	for _, r := range got.Rows {
		rows[r.Account+"/"+r.Currency] = r
	}
	require.Equal(t, "100", rows["Assets:Bank/CNY"].NativeAmount)
	require.Equal(t, "-20", rows["Assets:Other/CNY"].NativeAmount)
	require.Equal(t, "0", rows["Assets:Zero/CNY"].NativeAmount)
	require.Equal(t, "excluded", rows["Assets:Bank:Fixed/CNY"].Kind)
	require.Equal(t, "mixed", rows["Assets:Mixed/CNY"].Kind)
	require.Nil(t, rows["Assets:Mixed/CNY"].ReferenceAmount)
	require.Equal(t, "unclassified", rows["Assets:New/CNY"].Kind)
	require.Equal(t, "unclassified", rows["Assets:Wallet/USDT"].Kind)
	require.NotContains(t, rows, "Assets:Broker/GOOG")
	require.Equal(t, "2024-01-01", rows["Assets:Broker/USD"].QuoteDate)
	before := computeCashBalance(ps, quotes, rules, "CNY", cashDay("2024-01-01"))
	require.Equal(t, "0", before.ReferenceTotal)
	require.False(t, before.ClassificationComplete)
}
func TestCashBalanceMissingOrInvalidFXRetainsNative(t *testing.T) {
	ps := []posting.Posting{cashPosting("Assets:Bank", "USD", "10", "2024-01-01")}
	rules := []config.CashAccount{cashRule("Assets:Bank", "USD", "cash")}
	for _, test := range []struct {
		name   string
		quotes []price.Price
	}{
		{"missing", nil}, {"future", []price.Price{{CommodityName: "USD", Date: cashDay("2024-01-03"), Value: decimal.NewFromInt(7)}}},
		{"zero", []price.Price{{CommodityName: "USD", Date: cashDay("2024-01-01"), Value: decimal.Zero}}},
		{"negative", []price.Price{{CommodityName: "USD", Date: cashDay("2024-01-01"), Value: decimal.NewFromInt(-1)}}},
		{"new invalid supersedes old", []price.Price{{CommodityName: "USD", Date: cashDay("2024-01-01"), Value: decimal.NewFromInt(7)}, {CommodityName: "USD", Date: cashDay("2024-01-02"), Value: decimal.Zero}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := computeCashBalance(ps, test.quotes, rules, "CNY", cashDay("2024-01-02"))
			require.False(t, got.FXComplete)
			require.Equal(t, "10", got.Rows[0].NativeAmount)
			require.Nil(t, got.Rows[0].ReferenceAmount)
		})
	}
}
func TestCashBalanceTransferAndExpiredRules(t *testing.T) {
	rules := []config.CashAccount{cashRule("Assets:A", "CNY", "cash"), cashRule("Assets:B", "CNY", "cash")}
	rules[0].EffectiveUntil = "2024-01-03"
	ps := []posting.Posting{cashPosting("Assets:A", "CNY", "100", "2024-01-01"), cashPosting("Assets:A", "CNY", "-40", "2024-01-02"), cashPosting("Assets:B", "CNY", "40", "2024-01-02")}
	got := computeCashBalance(ps, nil, rules, "CNY", cashDay("2024-01-02"))
	require.Equal(t, "100", got.ReferenceTotal)
	got = computeCashBalance(ps, nil, rules, "CNY", cashDay("2024-01-03"))
	require.Equal(t, "40", got.ReferenceTotal)
	require.False(t, got.ClassificationComplete)
}
func TestCashBalanceAPIRejectsDatesAndDatabaseFailure(t *testing.T) {
	utils.SetNow("2024-01-02")
	t.Cleanup(utils.ResetNow)
	db := inMemoryDB(t)
	r := gin.New()
	r.GET("/api/cash_balance", CashBalanceHandler(db))
	for _, date := range []string{"bad", "2024-02-30", "2024-01-03"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/cash_balance?date="+date, nil))
		require.Equal(t, 400, w.Code)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/cash_balance", nil))
	require.Equal(t, 200, w.Code)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, false, got["classification_complete"])
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/cash_balance", nil))
	require.Equal(t, 500, w.Code)
}

func TestCashBalanceUnknownBankCurrenciesAreVisible(t *testing.T) {
	rules := []config.CashAccount{cashRule("Assets:Bank:Known", "CNY", "cash"), cashRule("Assets:CustomBank", "CNY", "cash")}
	ps := []posting.Posting{cashPosting("Assets:Bank:New", "USDT", "500", "2024-01-01"), cashPosting("Assets:CustomBank", "UNRECOGNIZED", "1", "2024-01-01")}
	got := computeCashBalance(ps, nil, rules, "CNY", cashDay("2024-01-02"))
	require.False(t, got.ClassificationComplete)
	pending := map[string]bool{}
	for _, row := range got.Rows {
		if row.Kind == "unclassified" {
			pending[row.Account+"/"+row.Currency] = true
			require.Nil(t, row.ReferenceAmount)
		}
	}
	require.True(t, pending["Assets:Bank:New/USDT"])
	require.True(t, pending["Assets:CustomBank/UNRECOGNIZED"])
	require.Equal(t, "0", got.ReferenceTotal)
}

func TestCashBalanceAPIUsesConfiguredCalendarDay(t *testing.T) {
	previous := config.GetConfig()
	previousLocal := time.Local
	time.Local = time.FixedZone("server-minus12", -12*60*60)
	config.SetConfigForTest(config.Config{DefaultCurrency: "CNY", TimeZone: "Pacific/Kiritimati"})
	utils.ResetNow()
	t.Cleanup(func() { time.Local = previousLocal; config.SetConfigForTest(previous); utils.ResetNow() })
	today := time.Now().In(config.TimeZone()).Format("2006-01-02")
	db := inMemoryDB(t)
	router := gin.New()
	router.GET("/api/cash_balance", CashBalanceHandler(db))
	for _, suffix := range []string{"", "?date=" + today} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/api/cash_balance"+suffix, nil))
		require.Equal(t, 200, response.Code)
		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &got))
		require.Equal(t, today, got["as_of"])
		require.Equal(t, today, got["today"])
	}
}
