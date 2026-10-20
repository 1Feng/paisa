# Cash balance generality implementation plan

> **For agentic workers:** Use superpowers:executing-plans to implement this approved minimal scope.

**Goal:** Keep explicit book-cash correctness while preserving unconfigured users and removing personal account-name defaults.
**Architecture:** Retain the independent cash endpoint and panel. Exact cash rules opt in; their accounts plus optional exact cash_review_accounts define review scope. Declared stock/fund commodities are excluded without per-position rules. Unclassified commodities never enter cash. Explicit cash rules cannot override declared securities. Old location values remain readable; new configs use bank/broker/other.
**Tech Stack:** Go, Svelte, existing YAML/JSON schema and Bun.
**Spec:** User-approved generality audit in this chat (2026-10-20).

## Global constraints
Preserve existing dirty files, local ledger and service. No merge. Actual cash income and overall redesign remain held. No payroll or quotation-notice changes in PR80. Omitted dates apply without a historical boundary; dates/evidence/access/location are optional. Explicit verified available still requires verified_on. No on-disk config rewrite. Do not use inherited account-prefix classifications.

## Review focus
- Unconfigured dashboard retains old checking cards.
- Arbitrary names and non-CNY base work without core rules.
- New unknown commodity inside configured scope stays pending, outside scope stays out.
- Declared securities cannot enter cash even with erroneous cash rule.
- Legacy cash configs remain readable and keep values.

## Task 1: Config and calculation
- [x] RED: minimal config, explicit Bridge/PPA, exact scope and metadata security tests.
- [x] Implement optional fields, exact cash_review_accounts, remove prefix guesses and enforced exclusions; declared securities excluded using existing commodities metadata.
- [x] GREEN: targeted config/server tests including history, precision, failures.

## Task 2: Compatible UI and documentation
- [x] Restore old checking cards when no rules/scope; show optional panel only on opt-in.
- [x] Use neutral institution groups, retain legacy group labels; simplify optional evidence display and scope wording.
- [x] Document minimal config, scope limits and local migration; retain prior plan as history, superseded here.

## Task 3: Acceptance and PR
- [x] Full Go/JS/check/format/build/regression validation, no automatic snapshot regeneration.
- [x] Browser verify opt-in, legacy fallback, pending/security/FX states; compare real ledger and preserved hashes.
- [ ] Commit scoped changes, update PR80, wait CI green then independent review; no merge.
