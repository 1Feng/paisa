# C01 Cash Account Balances Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan inline, then independently review the whole branch.

**Goal:** Identify cash with explicit account/currency rules and display native balances, dated reference values and coverage limits.
**Architecture:** Add an independent, read-only cash balance API and shared Svelte cash panel on the dashboard and asset balance page. Leave legacy income, cash flow, budget, investment, costs and valuation calculations unchanged. Cash classification does not reuse asset-kind defaults.
**Tech Stack:** Go/GORM/decimal, JSON schema/YAML, Svelte/TypeScript, existing journal prices.
**Spec:** User-approved C01 rules in this conversation, 2026-10-20; reproduced below.

## Global Constraints
- Exact account + ISO currency matching; normalize posting currency symbols; no inheritance, globs or automatic inclusion.
- Every rule has inclusive effective_from and optional exclusive effective_until, evidence and access conditions; overlapping rules fail validation.
- Native balances sum actual, nonforecast asset posting quantities through the selected valuation day. Rule dates classify the balance at that date, not which historical postings count toward it.
- Zero configured accounts remain visible; negative balances retain their sign. New/unconfigured cash candidates appear for review.
- Missing or nonpositive FX never becomes zero value; original currency remains visible. Only quotes on or before the valuation day qualify.
- Cash reference total is not spendable money. Location and withdrawal access are independent dimensions.
- Bridge and PPA cannot be included as cash. HSBC-CN remains mixed with no fixed amount deduction or manufactured split.
- Preserve all existing FinData edits; do not change journals or costs. Real account rules stay in ignored mydata; only synthetic config examples enter Git.
- New API monetary values are decimal strings. Report separate classification and conversion completeness and all pending/mixed omissions.
- Actual cash income and the broader reporting redesign remain held. No merge.

## Review Focus
- Mixed-currency broker accounts must not include securities (test exact USD rule vs GOOG/SGOV).
- Parent/child accounts cannot double count (test parent own postings and excluded fixed child).
- Rule changes cannot rewrite pre-effective history (test dated classification including full prior balance).
- Unknown currencies/accounts, missing quotes and API failures must not appear as complete zero (test empty policy, future FX and UI failure state).
- Zero/negative amounts and internal transfer conserve native money (test signed quantities, zero rows and equal/opposite cash transfer).

### Task 1: Policy + balance API
Files: internal/config/cash.go, cash_test.go, config.go, schema.json; internal/server/cash_balance.go, cash_balance_test.go, server.go.
- [x] Add failing config load tests for exact rules, overlapping periods, invalid dates/enums/currencies and forbidden cash inclusions. Run config tests RED.
- [x] Implement typed cash_accounts policy and validation; run config tests GREEN.
- [x] Add failing server tests for exact pair classification, nonfuture quotes, partial totals, dated rules, zeros, negatives and database failures. Run RED.
- [x] Implement GetCashBalance(db, asOf) returning native rows, reference subtotal, grouped subtotals, classification/FX completeness, rule provenance, latest posting and last verified dates. Register GET /api/cash_balance?date=YYYY-MM-DD with invalid/future dates rejected.
- [x] Run affected Go tests GREEN and commit.

### Task 2: Shared UI + documentation
Files: src/lib/cash_balance.ts; src/lib/components/CashBalances.svelte; dashboard and assets/balance pages; en/zh-CN locales; docs cash configuration guide.
- [x] Build one shared panel that fetches the additive API, shows date selector, location groups, exact original-currency balances, quote date, access/last-verified information, exclusions and mixed/pending accounts. Explicitly warn about missing configuration, missing FX and data freshness; never display a misleading zero on load failure.
- [x] Replace dashboard's legacy checking-card rendering with the panel; add same panel to asset balance page. Leave old APIs/snapshots unchanged.
- [x] Document exact matching, temporal rules, partial totals and synthetic config example. Format, type-check and build.

### Task 3: Acceptance + PR
- [ ] Run Go suite and JS/type/build checks appropriate to the change. Normal regression must not regenerate snapshots.
- [ ] Back up local config/SQLite and hash all preexisting FinData dirty files. Append approved cash rules to local config only, effective on today's confirmed scope; do not backdate unsupported classifications.
- [ ] Build loopback-only acceptance binary, serve against canonical FinData journal and existing cache, verify original balances, mixed exclusion, current FX, unchanged income/payroll/networth, browser dashboard/asset page, error logs and no future dates. Preserve backup and evidence locally.
- [ ] Independent whole-branch review, address material findings, commit/push and create PR. Attach PR, record CI accurately, do not merge.
