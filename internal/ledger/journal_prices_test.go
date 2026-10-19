package ledger

import (
	"encoding/json"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/stretchr/testify/require"
)

func journalPrice(t *testing.T, rows []price.Price, name, day string) price.Price {
	t.Helper()
	for _, p := range rows {
		if p.CommodityName == name && p.Date.Format("2006-01-02") == day {
			return p
		}
	}
	t.Fatalf("missing %s quote on %s", name, day)
	return price.Price{}
}

func TestJournalPricesForeignQuotesAndFXOnlyChanges(t *testing.T) {
	input := "P 2026/10/13 16:00:00 GOOG $349.52\n" +
		"P 2026/10/14 16:00:00 \"03690.HK\" 77.1 HKD\n" +
		"P 2026/10/13 00:00:00 $ 6.692 ¥\n" +
		"P 2026/10/13 00:00:00 HKD 0.852758 ¥\n" +
		"P 2026/10/15 00:00:00 USD 7 CNY\n"
	rows, err := parseLedgerPrices(input, "CNY")
	require.NoError(t, err)
	require.Equal(t, "2338.98784", journalPrice(t, rows, "GOOG", "2026-10-13").Value.String())
	require.Equal(t, "2446.64", journalPrice(t, rows, "GOOG", "2026-10-15").Value.String())
	require.Equal(t, "65.7476418", journalPrice(t, rows, "03690.HK", "2026-10-14").Value.String())
}

func TestJournalPricesDoNotUseFutureFX(t *testing.T) {
	rows, err := parseLedgerPrices("P 2024/01/01 16:00:00 TEST $100\nP 2024/01/02 00:00:00 USD 7 CNY\n", "CNY")
	require.NoError(t, err)
	p := journalPrice(t, rows, "TEST", "2024-01-01")
	b, err := json.Marshal(p)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(b, &payload))
	require.Equal(t, "missing_fx", payload["conversion"].(map[string]any)["status"])
	require.Equal(t, "700", journalPrice(t, rows, "TEST", "2024-01-02").Value.String())
}

func TestJournalPricesStrictReverseAndPivot(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"reverse", "P 2024/01/01 00:00:00 CNY 0.125 USD\nP 2024/01/01 16:00:00 TEST $100\n", "800"},
		{"pivot", "P 2024/01/01 00:00:00 HKD 0.128 USD\nP 2024/01/01 00:00:00 USD 7.2 CNY\nP 2024/01/01 16:00:00 TEST 100 HKD\n", "92.16"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := parseLedgerPrices(tt.input, "CNY")
			require.NoError(t, err)
			require.Equal(t, tt.want, journalPrice(t, rows, "TEST", "2024-01-01").Value.String())
		})
	}
}

func TestJournalPricesDailyOrderAndExplicitBasePrice(t *testing.T) {
	a := "P 2024/01/01 16:00:00 TEST $100\nP 2024/01/01 00:00:00 USD 7 CNY\nP 2024/01/01 16:00:00 TEST 650 CNY\n"
	b := "P 2024/01/01 16:00:00 TEST 650 CNY\nP 2024/01/01 00:00:00 USD 7 CNY\nP 2024/01/01 16:00:00 TEST $100\n"
	for _, input := range []string{a, b} {
		rows, err := parseLedgerPrices(input, "CNY")
		require.NoError(t, err)
		require.Equal(t, "650", journalPrice(t, rows, "TEST", "2024-01-01").Value.String())
	}
}

func TestJournalPricesCoverageBoundaries(t *testing.T) {
	for _, tt := range []struct{ name, input, day, want, status string }{
		{"reverse future", "P 2024/01/01 16:00:00 TEST $100\nP 2024/01/02 00:00:00 CNY 0.125 USD\n", "2024-01-01", "0", "missing_fx"},
		{"pivot future", "P 2024/01/01 00:00:00 HKD 0.128 USD\nP 2024/01/02 00:00:00 USD 7.2 CNY\nP 2024/01/01 16:00:00 TEST 100 HKD\n", "2024-01-01", "0", "missing_fx"},
		{"zero FX", "P 2024/01/01 00:00:00 USD 0 CNY\nP 2024/01/01 16:00:00 TEST $100\n", "2024-01-01", "0", "missing_fx"},
		{"negative FX", "P 2024/01/01 00:00:00 USD -7 CNY\nP 2024/01/01 16:00:00 TEST $100\n", "2024-01-01", "0", "missing_fx"},
		{"zero security", "P 2024/01/01 00:00:00 USD 7 CNY\nP 2024/01/01 16:00:00 TEST $0\n", "2024-01-01", "0", "converted"},
		{"latest native time", "P 2024/01/01 17:00:00 TEST $102\nP 2024/01/01 00:00:00 USD 7 CNY\nP 2024/01/01 16:00:00 TEST $100\n", "2024-01-01", "714", "converted"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := parseLedgerPrices(tt.input, "CNY")
			require.NoError(t, err)
			p := journalPrice(t, rows, "TEST", tt.day)
			require.Equal(t, tt.want, p.Value.String())
			require.Equal(t, tt.status, p.Conversion.Status)
		})
	}
}

func TestJournalPricesRejectConflictingSameTimeFX(t *testing.T) {
	_, err := parseLedgerPrices("P 2024/01/01 00:00:00 USD 7 CNY\nP 2024/01/01 00:00:00 $ 8 ¥\nP 2024/01/01 16:00:00 TEST $100\n", "CNY")
	require.Error(t, err)
}

func TestJournalPricesDirectBaseUsesLatestNativeTime(t *testing.T) {
	early := "P 2024/01/01 09:00:00 TEST 700 CNY\n"
	late := "P 2024/01/01 17:00:00 TEST 710 CNY\n"
	for _, input := range []string{early + late, late + early} {
		rows, err := parseLedgerPrices(input, "CNY")
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, "710", rows[0].Value.String())
	}
}

func TestJournalPricesInvalidCurrencyRateIsUnavailable(t *testing.T) {
	for _, value := range []string{"0", "-7"} {
		rows, err := parseLedgerPrices("P 2024/01/01 00:00:00 USD "+value+" CNY\n", "CNY")
		require.NoError(t, err)
		require.False(t, journalPrice(t, rows, "USD", "2024-01-01").Available())
	}
	for _, security := range []string{"TEST", "AMD"} {
		rows, err := parseLedgerPrices("P 2024/01/01 00:00:00 "+security+" 0 CNY\n", "CNY")
		require.NoError(t, err)
		require.True(t, rows[0].Available(), "ticker alone must not identify an FX node")
	}
}
