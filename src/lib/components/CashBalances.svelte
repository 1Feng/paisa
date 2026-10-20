<script lang="ts">
  import { onMount } from "svelte";
  import BigNumber from "bignumber.js";
  import { fetchCashBalance, type CashBalance } from "$lib/cash_balance";
  import { _ as t } from "$lib/i18n";

  let date = "";
  let report: CashBalance;
  let loading = true;
  let failed = false;
  let request = 0;
  const locations = ["bank", "domestic_bank", "overseas_bank", "broker", "other"];
  const amount = (value: string) => new BigNumber(value).toFormat();
  const reference = (value: string) => new BigNumber(value).toFormat(2);
  async function load() {
    const id = ++request;
    loading = true;
    failed = false;
    try {
      const next = await fetchCashBalance(date || undefined);
      if (id === request) {
        report = next;
        date = next.as_of;
      }
    } catch {
      if (id === request) failed = true;
    } finally {
      if (id === request) loading = false;
    }
  }
  onMount(load);
</script>

<section class="box cash-balances" aria-label={$t("cash.title")}>
  <div
    class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap mb-3"
  >
    <h2 class="title is-5 mb-2">{$t("cash.title")}</h2>
    <label class="is-size-7">
      {$t("cash.as_of")}
      <input
        class="input is-small"
        type="date"
        bind:value={date}
        max={report?.today}
        on:change={load}
      />
    </label>
  </div>
  <p class="is-size-7 mb-3">{$t("cash.boundary")}</p>
  {#if loading}
    <p role="status">{$t("cash.loading")}</p>
  {:else if failed}
    <div class="notification is-danger is-light" role="alert">
      {$t("cash.failed")}
      <button class="button is-small ml-2" on:click={load}>{$t("cash.retry")}</button>
    </div>
  {:else if report}
    {#if !report.policy_configured}
      <p class="notification is-warning is-light">{$t("cash.not_configured")}</p>
    {/if}
    <p class="is-size-7 has-text-grey mb-3">
      {$t("cash.last_posting")}: {report.last_posting || $t("cash.unknown")} ·
      {$t("cash.sync")}: {report.last_journal_sync || $t("cash.not_recorded")}
      <br />{$t("cash.freshness")}
    </p>
    <div class="notification is-light">
      <strong
        >{$t("cash.reference_total")}: {report.currency} {reference(report.reference_total)}</strong
      >
      <p class="is-size-7">{$t(report.fx_complete ? "cash.fx_complete" : "cash.fx_partial")}</p>
      <p class="is-size-7">
        {$t(report.classification_complete ? "cash.classified" : "cash.classification_partial")}
      </p>
    </div>
    {#each locations.filter( (location) => report.rows.some((r) => r.kind === "cash" && r.location === location) ) as location}
      <h3 class="subtitle is-6 mt-4 mb-2">
        {$t(`cash.location.${location}`)} · {report.currency}
        {reference(report.groups[location].reference_total)}
        {#if !report.groups[location].fx_complete}
          · {$t("cash.fx_partial_short")}{/if}
      </h3>
      <div class="table-container">
        <table class="table is-fullwidth is-size-7">
          <thead
            ><tr
              ><th>{$t("cash.account")}</th><th>{$t("cash.native")}</th><th
                >{$t("cash.reference")}</th
              ><th>{$t("cash.conditions")}</th></tr
            ></thead
          >
          <tbody>
            {#each report.rows.filter((r) => r.kind === "cash" && r.location === location) as row}
              <tr>
                <td
                  ><span class="account">{row.account}</span>
                  {#if row.rule?.evidence || row.rule?.effective_from || row.rule?.effective_until}
                    <details>
                      <summary>{$t("cash.rule")}</summary>
                      <p>{row.rule?.evidence}</p>
                      <p>
                        {row.rule?.effective_from || $t("cash.open_start")} — {row.rule
                          ?.effective_until || $t("cash.open_ended")} ({$t("cash.exclusive")})
                      </p>
                    </details>
                  {/if}
                </td>
                <td class:has-text-danger={new BigNumber(row.native_amount).isNegative()}
                  >{row.currency}
                  {amount(row.native_amount)}
                  {#if new BigNumber(row.native_amount).isNegative()}<p>
                      {$t("cash.negative")}
                    </p>{/if}
                </td>
                <td
                  >{#if row.reference_amount === null}<span class="has-text-danger"
                      >{$t("cash.missing_fx")}</span
                    >{:else}{report.currency} {reference(row.reference_amount)}{/if}
                  {#if row.quote_date}<p>{$t("cash.fx_date")}: {row.quote_date}</p>{/if}
                </td>
                <td
                  >{$t(`cash.access.${row.access}`)}<br />{$t("cash.last_posting")}: {row.last_posting ||
                    $t("cash.unknown")}<br />{$t("cash.verified")}: {row.rule?.verified_on ||
                    $t("cash.unknown")}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/each}
    <details class="mt-4" open={!report.classification_complete}>
      <summary
        >{$t("cash.pending")} ({report.rows.filter(
          (r) => r.kind === "mixed" || r.kind === "unclassified"
        ).length})</summary
      >
      <p class="is-size-7 my-2">{$t("cash.pending_help")}</p>
      <ul class="is-size-7">
        {#each report.rows.filter((r) => r.kind === "mixed" || r.kind === "unclassified") as row}
          <li class="mb-2">
            <span class="account">{row.account}</span> · {row.currency}
            {amount(row.native_amount)} · {$t(`cash.kind.${row.kind}`)}
            <p>{row.rule?.evidence || $t("cash.no_rule")}</p>
          </li>
        {/each}
      </ul>
    </details>
    <details class="mt-3">
      <summary
        >{$t("cash.excluded")} ({report.rows.filter((r) => r.kind === "excluded").length})</summary
      >
      <ul class="is-size-7">
        {#each report.rows.filter((r) => r.kind === "excluded") as row}
          <li class="my-2">
            <span class="account">{row.account}</span> · {row.currency}
            {amount(row.native_amount)}<br />{row.rule?.evidence || $t("cash.excluded_rule")}
          </li>
        {/each}
      </ul>
    </details>
  {/if}
</section>

<style>
  .cash-balances {
    min-width: 0;
  }
  .account {
    overflow-wrap: anywhere;
  }
  td {
    min-width: 140px;
  }
  td:first-child {
    min-width: 210px;
  }
  summary {
    cursor: pointer;
  }
  input {
    width: auto;
  }
</style>
