package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSalaryIncomeAccountsConfigRoundTrip(t *testing.T) {
	previous := GetConfig()
	t.Cleanup(func() { SetConfigForTest(previous) })
	for _, tc := range []struct {
		name, setting string
		want          any
		present       bool
	}{
		{"legacy omitted", "", nil, false},
		{"custom accounts", "salary_income_accounts: [Income:Employment:Wages, Income:Employment:Bonus]\n", []any{"Income:Employment:Wages", "Income:Employment:Bonus"}, true},
		{"explicit empty", "salary_income_accounts: []\n", []any{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, LoadConfig([]byte(baseConfig+tc.setting), ""))
			for _, format := range []string{"yaml", "json"} {
				var result map[string]any
				if format == "yaml" {
					encoded, err := yaml.Marshal(GetConfig())
					require.NoError(t, err)
					require.NoError(t, yaml.Unmarshal(encoded, &result))
					require.NoError(t, LoadConfig(encoded, ""))
				} else {
					encoded, err := json.Marshal(GetConfig())
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(encoded, &result))
					require.NoError(t, LoadConfig(encoded, ""))
				}
				value, present := result["salary_income_accounts"]
				assert.Equal(t, tc.present, present, format)
				assert.Equal(t, tc.want, value, format)
			}
		})
	}
}

func TestSalaryIncomeAccountsConfigValidation(t *testing.T) {
	previous := GetConfig()
	t.Cleanup(func() { SetConfigForTest(previous) })
	for _, setting := range []string{
		"null", "Income:Salary", "[123]", "['']", "[Income:Salary, Income:Salary]", "[Assets:Employment]", "[Income:Employment:]", "[Income:Employment::Salary]", "[Income:Employment:*]",
	} {
		t.Run(setting, func(t *testing.T) {
			assert.Error(t, LoadConfig([]byte(baseConfig+"salary_income_accounts: "+setting+"\n"), ""))
		})
	}
}
