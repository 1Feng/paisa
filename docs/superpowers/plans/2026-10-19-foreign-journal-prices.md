# Foreign journal prices implementation plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan task by task.

**Goal:** Restore Ledger foreign-denominated quotes, value them strictly as of each historical date, and expose missing FX coverage.

**Architecture:** Convert journal quotes into existing default-currency price rows at quote/FX change dates. Persist optional conversion provenance, including unresolved points. Keep posting amounts and the existing process-wide FX store unchanged. Add read-only coverage diagnostics and a shared notice, while the price page shows unresolved quotes and provenance.

**Tech stack:** Go, shopspring/decimal, GORM SQLite, SvelteKit.

**Spec:** User-approved scope in this chat on 2026-10-19: restore foreign quotes, strict historical conversion and gap notices; historical costs separately; broader reporting redesign remains on hold; preserve local changes; verify data/UI and open an unmerged PR.

## Global constraints
- FinData remains the sole journal; never edit its files or fetch replacement market data.
- No posting cost conversion, income/cash classification, or expense-chart changes.
- No future rates, including inversion and USD-pivot legs. Keep the existing FX store semantics for other consumers.
- Preserve effective daily default-currency price values and existing API fields. New provenance is optional.
- Run ordinary snapshots read-only; only explicitly regenerate justified changed expectations.

## Review focus
- FX-only changes must revalue an unchanged native security quote.
- Missing early FX must remain missing until the first usable rate, rather than leaking future information.
- A security ticker that is also a currency code must not become an FX intermediary unless used as a quote denomination or already recognized by the existing currency-symbol aliases.
- Explicit zero price must differ from missing quote; failed sync must preserve existing prices.
- Same-day native timestamps and direct-default-currency precedence must be deterministic; reordered input must give identical results.

## Task 1: Quote conversion (RED → GREEN)
Files: internal/ledger/journal_prices.go, internal/ledger/ledger.go, internal/model/price/price.go, internal/ledger/journal_prices_test.go.
- [x] Write behavior tests through parseLedgerPrices, confirm RED for USD/HKD quotes and FX-only changes.
- [x] Parse native rows first; convert with strict direct/reverse/USD lookup on daily dates. Sort original timestamps before daily coalescing. Emit optional conversion metadata and missing-FX rows. Keep native commodity names.
- [x] Test missing/future FX, zero/invalid FX, explicit zero security prices, reordered rows, digit/dotted tickers, base-currency precedence and alias handling.

## Task 2: Persistence and coverage (RED → GREEN)
Files: internal/model/model.go, internal/service/market.go, internal/server/valuation.go, internal/server/server.go and corresponding tests.
- [x] Test real SQLite roundtrip, zero-value lookup, missing coverage and preservation after parse failures.
- [x] Parse postings before any replacement. Commit price/posting replacements atomically using existing model helpers where possible.
- [x] Add /api/valuation with historical gaps and current missing holdings. Missing records must not become valid converted prices. Existing legacy numerical fallbacks remain compatibility estimates and are explicitly flagged; UI must not label incomplete historical figures reliable.

## Task 3: UI
Files: price page, shared ValuationNotice component, app layout, Price typing and i18n.
- [x] Show native price/currency, converted price and FX/quote dates; retain missing rows.
- [x] Display shared coverage notice with dates and link to price details. State that restoring prices does not validate historical costs or returns.

## Task 4: Validation, PR and independent review
- [x] Run focused and full Go/JS/regression/lint checks; verify ordinary tests do not rewrite snapshots.
- [x] Sync a temporary DB from FinData and compare native quantities, posting amounts, 10 restored prices, historical gaps, balance identities and repeat sync/restart behavior.
- [x] Back up local SQLite, rebuild loopback service, sync journal only, inspect browser prices, coverage notice and affected asset pages.
- [x] Verify preserved FinData changes and expense chart. Independent review corrected daily-order and invalid-FX edge cases and approved the final diff.
- [ ] Commit only scoped files, push PR, attach it and record independent review against the final HEAD. Do not merge.

## Acceptance evidence (2026-10-19)
- `make test` and `make lint` passed; after final reviewer fixes, `go test ./...` and `make jstest` passed again. Ordinary regression verification left all 44 snapshot files and their mtimes unchanged.
- Repeated journal-only synchronization into SQLite and a service restart retained posting facts, quantities and cost amounts. Existing base-price rows coalesce to 495 effective daily points with the same effective values.
- Ten affected current securities match native quote multiplied by the applicable historical FX rate. All conversion legs are dated on or before their valuation day. Fourteen unresolved historical points remain explicitly flagged after restart.
- Browser acceptance on the loopback local service covered quote provenance and expanded gaps, asset balances, net worth, allocation, gains, investment/payroll and monthly expenses. No browser errors were observed on the final service.
- Local backups and acceptance artifacts are ignored under `mydata/`; no personal journal, portfolio data or screenshots belong in this PR.
- Price history operates at daily end-of-day granularity. Same-day quote ordering uses native timestamps; direct-base quotes win ties. Compatibility fallback estimates remain in existing monetary APIs when coverage is missing and are explicitly warned about. Historical cost conversion and the broader redesign remain deferred.
