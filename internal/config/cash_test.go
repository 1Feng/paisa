package config

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCashPolicyLoadAndRejectInvalidRules(t *testing.T) {
	prev := GetConfig()
	t.Cleanup(func() { SetConfigForTest(prev) })
	rule := `cash_accounts:
  - account: Assets:Bank
    currency: USD
    kind: cash
    location: overseas_bank
    access: conditional
    effective_from: "2026-10-20"
    evidence: Confirmed statement
`
	require.NoError(t, LoadConfig([]byte(baseConfig+rule), ""))
	for _, tc := range []struct{ name, from, to string }{
		{"wildcard", "Assets:Bank", "Assets:Bank:*"},
		{"bad date", "2026-10-20", "2026-02-30"},
		{"not fiat", "currency: USD", "currency: USDT"},
		{"available needs verification", "access: conditional", "access: available"},
		{"location", "overseas_bank", "mars"},
		{"access", "conditional", "unlimited"},
		{"inverted", "effective_from: \"2026-10-20\"", "effective_from: \"2026-10-20\"\n    effective_until: \"2026-10-19\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, LoadConfig([]byte(baseConfig+strings.ReplaceAll(rule, tc.from, tc.to)), ""))
		})
	}
	// Adjacent half-open intervals are valid; overlap is rejected regardless of kind.
	extra := `  - account: Assets:Bank
    currency: USD
    kind: mixed
    location: overseas_bank
    access: unknown
    effective_from: "2026-10-21"
    evidence: Account now mixed
`
	require.Error(t, LoadConfig([]byte(baseConfig+rule+extra), ""))
	adjacent := strings.Replace(rule, "    evidence:", "    effective_until: \"2026-10-21\"\n    evidence:", 1)
	require.NoError(t, LoadConfig([]byte(baseConfig+adjacent+extra), ""))
	require.NoError(t, LoadConfig([]byte(baseConfig+strings.ReplaceAll(strings.ReplaceAll(rule, "kind: cash", "kind: excluded"), "currency: USD", "currency: USDT")), ""))
	require.NoError(t, LoadConfig([]byte(baseConfig), ""), "existing configurations remain valid")
}

func TestCashMinimalRulesAndNeutralGroups(t *testing.T) {
	prev := GetConfig()
	t.Cleanup(func() { SetConfigForTest(prev) })
	for _, account := range []string{"Assets:Everyday", "Assets:Bridge:Ordinary", "Assets:PPA:Ordinary"} {
		require.NoError(t, LoadConfig([]byte(baseConfig+`cash_accounts:
  - account: `+account+`
    currency: EUR
    kind: cash
`), ""))
	}
	require.NoError(t, LoadConfig([]byte(baseConfig+`cash_accounts:
  - account: Assets:Everyday
    currency: EUR
    kind: cash
    location: bank
cash_review_accounts: [Assets:Investing]
`), ""))
}

func TestCashOptionalIntervalsAndReviewScopeValidation(t *testing.T) {
	prev := GetConfig()
	t.Cleanup(func() { SetConfigForTest(prev) })
	for _, bad := range []string{"Assets:Investing:*", "Liabilities:Card", " Assets:Bank"} {
		require.Error(t, LoadConfig([]byte(baseConfig+"cash_review_accounts: [\""+bad+"\"]\n"), ""))
	}
	minimal := `cash_accounts:
  - account: Assets:Everyday
    currency: EUR
    kind: cash
    effective_until: "2024-01-03"
  - account: Assets:Everyday
    currency: EUR
    kind: excluded
    effective_from: "2024-01-03"
`
	require.NoError(t, LoadConfig([]byte(baseConfig+minimal), ""))
	require.Error(t, LoadConfig([]byte(baseConfig+strings.ReplaceAll(minimal, "effective_from: \"2024-01-03\"", "effective_from: \"2024-01-02\"")), ""))
}

func TestMinimalCashConfigurationCanBeSavedAndReloaded(t *testing.T) {
	prev := GetConfig()
	t.Cleanup(func() { SetConfigForTest(prev) })
	require.NoError(t, LoadConfig([]byte(baseConfig+`cash_accounts:
  - account: Assets:Everyday
    currency: EUR
    kind: cash
`), ""))
	data, err := json.Marshal(GetConfig())
	require.NoError(t, err)
	require.NoError(t, LoadConfig(data, ""), "config editor JSON save must preserve minimal rules")
}
