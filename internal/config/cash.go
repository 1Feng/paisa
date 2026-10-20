package config

import (
	"fmt"
	"golang.org/x/text/currency"
	"strings"
	"time"
)

// CashAccount is an explicit classification of one exact account/currency pair.
// Dates classify the whole balance at a valuation day, not individual postings.
type CashAccount struct {
	Account        string `json:"account" yaml:"account"`
	Currency       string `json:"currency" yaml:"currency"`
	Kind           string `json:"kind" yaml:"kind"`
	Location       string `json:"location" yaml:"location"`
	Access         string `json:"access" yaml:"access"`
	EffectiveFrom  string `json:"effective_from" yaml:"effective_from"`
	EffectiveUntil string `json:"effective_until,omitempty" yaml:"effective_until,omitempty"`
	Evidence       string `json:"evidence" yaml:"evidence"`
	VerifiedOn     string `json:"verified_on,omitempty" yaml:"verified_on,omitempty"`
}

func (r CashAccount) Applies(day string) bool {
	return day >= r.EffectiveFrom && (r.EffectiveUntil == "" || day < r.EffectiveUntil)
}

func validateCashAccounts(rules []CashAccount) error {
	for i, r := range rules {
		if !strings.HasPrefix(r.Account, "Assets:") || strings.TrimSpace(r.Account) != r.Account || strings.ContainsAny(r.Account, "*?[]") {
			return fmt.Errorf("cash_accounts[%d]: an exact Assets account is required", i)
		}
		if _, err := currency.ParseISO(r.Currency); r.Kind == "cash" && (err != nil || r.Currency != strings.ToUpper(r.Currency) || r.Currency == "XXX" || r.Currency == "XTS") {
			return fmt.Errorf("cash_accounts[%d]: currency must be an ISO currency code", i)
		}
		if strings.TrimSpace(r.Evidence) == "" {
			return fmt.Errorf("cash_accounts[%d]: evidence is required", i)
		}
		if r.Access == "available" && r.VerifiedOn == "" {
			return fmt.Errorf("cash_accounts[%d]: available access requires verified_on", i)
		}
		if strings.TrimSpace(r.Currency) != r.Currency || strings.ContainsAny(r.Currency, "*?[]") {
			return fmt.Errorf("cash_accounts[%d]: exact currency or commodity required", i)
		}
		for _, date := range []string{r.EffectiveFrom, r.EffectiveUntil, r.VerifiedOn} {
			if date == "" {
				continue
			}
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return fmt.Errorf("cash_accounts[%d]: invalid date %q", i, date)
			}
		}
		if r.EffectiveUntil != "" && r.EffectiveUntil <= r.EffectiveFrom {
			return fmt.Errorf("cash_accounts[%d]: empty or reversed effective interval", i)
		}
		if r.Kind == "cash" && (r.Account == "Assets:Bridge" || strings.HasPrefix(r.Account, "Assets:Bridge:") || r.Account == "Assets:PPA" || strings.HasPrefix(r.Account, "Assets:PPA:")) {
			return fmt.Errorf("cash_accounts[%d]: transit and retirement accounts cannot be cash", i)
		}
		for j := 0; j < i; j++ {
			other := rules[j]
			if r.Account == other.Account && r.Currency == other.Currency && (r.EffectiveUntil == "" || other.EffectiveFrom < r.EffectiveUntil) && (other.EffectiveUntil == "" || r.EffectiveFrom < other.EffectiveUntil) {
				return fmt.Errorf("cash_accounts[%d]: overlapping rule for %s / %s", i, r.Account, r.Currency)
			}
		}
	}
	return nil
}
