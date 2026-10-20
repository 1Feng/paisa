# Optional book-cash balances

Paisa can show a separate book-cash panel on the dashboard and Assets → Balance.
Configure `cash_accounts` or `cash_review_accounts` to enable it. Without either,
the dashboard retains the existing checking-balance cards and the asset page is
unchanged. The feature does not change income, cash-flow, budget, investment,
historical-cost or existing asset-classification calculations.

## Minimal configuration

```yaml
cash_accounts:
  - account: Assets:Everyday
    currency: EUR
    kind: cash
  - account: Assets:Investing
    currency: USD
    kind: cash
    location: broker
```

Only `account`, `currency` and `kind` are required. Accounts and commodities match
exactly; parent rules do not classify descendants. Cash requires an ISO currency
code. Posting symbols such as `$` and `¥` are normalized; USDT is not USD and is
not fiat cash. Kinds are `cash` (included), `mixed` and `excluded` (not included).
An account name does not determine inclusion: transit, retirement, institution
and personal-account naming conventions belong in each user's configuration.

### Review scope and securities

Every account named by a cash rule is reviewed for other currencies/commodities.
Optionally add exact accounts without yet classifying their positions:

```yaml
cash_review_accounts:
  - Assets:Investing
  - Assets:Unreviewed
```

No other accounts are searched automatically. Newly created or renamed accounts
must be added to the scope. Completeness means **within the configured scope**,
not all household cash. There is no guarantee that every cash account was selected.

Existing `commodities[].type` declarations of `stock` or `mutualfund` exclude those
securities from cash in all reviewed accounts, without a separate rule per holding.
A conflicting cash rule cannot include a declared security. Tickers alone and
inherited account-name guesses are not proof of a security type. Unknown codes
remain pending and outside the subtotal; a new currency inside the review scope
cannot silently disappear or be guessed to be cash. Explicit `excluded` or `mixed`
rules can classify other non-fiat commodities. Confirmed security declarations
are current metadata, not a dated history of instrument classification.

## Optional detail and compatibility

- `location`: `bank`, `broker` or `other` (default). Legacy `domestic_bank` and
  `overseas_bank` values remain readable and retain their separate display groups;
  new users do not need to classify geography.
- `access`: `unknown` (default), `conditional`, or `available`. `available` requires
  `verified_on`. This describes withdrawal information and does not change book cash.
- `evidence`: optional explanatory note.
- `effective_from`: inclusive; omitted means no start limit.
- `effective_until`: exclusive; omitted means no end limit.
- `verified_on`: optional verification date, distinct from posting and sync dates.

Dated rules classify the whole balance on the selected day, including earlier
deposits. Inactive rules authorize no inclusion. Overlapping rules for the same
account/commodity and invalid dates or wildcard account names fail validation.
Old cash configurations remain readable without rewriting the on-disk YAML.
Specific transit, retirement and mixed-account decisions must be expressed as
local `excluded`/`mixed` rules. They are no longer imposed by core account prefixes.
Existing explicit local rules continue to express those decisions.

## Values and limitations

- Original amounts use posting quantities, not acquisition cost. Each posting
  contributes once to its exact account; internal transfers cancel in native sums.
- Negative balances retain their sign; configured zero balances remain visible.
- Reference values use journal FX on or before the selected day in
  `default_currency`, with no future quotes or backward extrapolation.
- Missing/invalid FX preserves the original amount and a null reference amount.
  Only converted balances enter the partial subtotal. Exact zero needs no FX.
- Unknown/mixed nonzero positions make classification incomplete. Classification
  and FX completeness are independent; neither proves bank reconciliation.
- Mixed positions are not split by fixed deduction or residual estimation.
- This is book cash, not spendable money. Debts, entrusted funds, settlement
  restrictions and planned spending are not deducted.
- External journal edits require the normal refresh. Unknown journal-sync time
  remains null rather than being inferred from transaction or file dates.

`GET /api/cash_balance?date=YYYY-MM-DD` returns both panels' data. Date defaults
to today in the configured time zone. Invalid/future dates return 400; database
failure returns 500 rather than an empty zero balance. Money uses decimal strings.
No external quotes are fetched by this endpoint.
