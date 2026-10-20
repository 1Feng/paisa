# Identified cash balances (C01)

The dashboard and Assets → Balance display a separate, read-only cash panel.
It reports **identified book cash**, not spendable money or a budget. Liabilities,
entrusted funds and planned spending are not deducted. It does not change the
income, cash-flow, budget, investment or historical-cost calculations.

Configure `cash_accounts` in Paisa's existing YAML settings. No rules means no
implicit cash inclusions. Review unclassified positions before using the subtotal.
Rules match **one exact account and currency**, not descendants. Included cash requires an ISO currency code; excluded/mixed rules may name an exact non-fiat commodity such as USDT. Symbols in
postings (for example `$` and `¥`) are normalized to their ISO currency. A broker's
USD balance does not include its securities. USDT is not USD and cannot be included
by a fiat-currency rule. A new account, renamed account or new currency requires review.

```yaml
cash_accounts:
  - account: Assets:Bank:Everyday
    currency: CNY
    kind: cash
    location: domestic_bank
    access: unknown
    effective_from: "2026-10-20"
    evidence: "Account scope reviewed; withdrawal availability not verified"
  - account: Assets:Broker:Combined
    currency: USD
    kind: cash
    location: broker
    access: conditional
    effective_from: "2026-10-20"
    evidence: "Fiat balance only; settlement and transfer conditions apply"
  - account: Assets:Bank:Everyday:Fixed
    currency: CNY
    kind: excluded
    location: domestic_bank
    access: conditional
    effective_from: "2026-10-20"
    evidence: "Fixed-term deposit, not current cash"
  - account: Assets:Bank:Mixed
    currency: CNY
    kind: mixed
    location: domestic_bank
    access: unknown
    effective_from: "2026-10-20"
    evidence: "Contains both a structured deposit and current funds; no supported split"
```

Locations: `domestic_bank`, `overseas_bank`, `broker`, `other`.
Access: `available` (confirmed at `verified_on`), `conditional` (settlement/transfer
conditions apply), `unknown`. Access does not change the book-cash subtotal.
`verified_on` is an optional YYYY-MM-DD date except when access is `available`.
It is distinct from the latest posting date and does not guarantee future availability.

Kinds: `cash` is included; `mixed` and `excluded` are not. Every rule requires an
explanation in `evidence`. Transit (`Assets:Bridge`) and retirement (`Assets:PPA`)
accounts cannot be configured as cash. Unclassified fiat positions and unfamiliar
bank/wallet commodities are shown for review; no security-price fallback is used.

Dates are calendar days in the configured time zone. `effective_from` is inclusive;
optional `effective_until` is exclusive. Conflicting/overlapping rules for a pair,
invalid dates and wildcard accounts fail configuration validation. Rules classify
the **whole balance at the selected date**: all prior nonforecast quantities are
included, even if deposited before the rule starts. An expired or not-yet-effective
rule does not authorize inclusion. Do not backdate classifications without evidence.

## Values and limitations

- Each posting contributes once to its own account/currency. Parent account rows
  do not include child balances. Internal cash transfers cancel in native sums.
- Negative balances are retained with a warning. Configured zero accounts remain visible.
- Original balances use quantities, not acquisition costs. Reference values use
  eligible journal FX quotes on or before the selected date in `default_currency`.
  Nonpositive FX, future quotes or missing conversion legs do not create values.
- Missing FX keeps the original amount and a null reference amount. Only converted
  positions enter the reference subtotal, labelled partial. Exactly zero balances
  need no FX to have an exactly zero reference value.
- Classification completeness and FX completeness are separate. Mixed/unclassified
  nonzero positions make identification incomplete, even if all included positions
  have FX. Full identification does not mean bank verification or spendability.
- Mixed accounts stay unsplit: no fixed deduction, residual-as-cash assumption or
  automatic backdated split. Resolve the source/classification separately.
- The existing cache has no journal-sync timestamp; the panel explicitly reports
  it as unrecorded. Posting date, bank verification date and sync time are different.
  External journal edits still require the normal journal refresh.

`GET /api/cash_balance?date=YYYY-MM-DD` returns the same data for both panels.
Date defaults to today in the configured time zone; `today` in the response also supplies the date picker maximum, independent of browser/server time zones. Invalid or future dates return 400. Database failures
return 500 instead of a successful empty balance. Money uses decimal strings;
`last_journal_sync` is null when unknown. No new external price fetch is performed.
