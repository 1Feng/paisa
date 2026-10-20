<script lang="ts">
  import { onMount } from "svelte";
  import { ajax, type ValuationCoverage } from "$lib/utils";
  import { _ as t } from "$lib/i18n";

  let coverage: ValuationCoverage;
  let failed = false;
  onMount(async () => {
    try {
      coverage = await ajax("/api/valuation", { background: true });
    } catch {
      failed = true;
    }
  });
</script>

{#if failed || coverage?.has_conversions || coverage?.current_missing.length}
  <aside class="notification is-warning is-light m-4" aria-label={$t("valuation.title")}>
    <strong>{$t("valuation.title")}</strong>
    {#if failed}
      <p>{$t("valuation.failed")}</p>
    {:else}
      {#if !coverage.current_complete}
        <p>{$t("valuation.current_missing")}: {coverage.current_missing.join(", ")}</p>
      {/if}
      {#if coverage.gaps.length}
        <p>{$t("valuation.incomplete")}</p>
        <details>
          <summary>{$t("valuation.gaps")} ({coverage.gaps.length})</summary>
          <ul>
            {#each coverage.gaps as gap}
              <li>
                {gap.commodity}: {gap.date.format("YYYY-MM-DD")}
                {#if gap.until_date}
                  — {gap.until_date.format("YYYY-MM-DD")} ({$t("valuation.exclusive")}){/if}
                · {gap.currency}
              </li>
            {/each}
          </ul>
        </details>
      {/if}
      <p>{$t("valuation.costs_deferred")}</p>
      <a href="/ledger/price">{$t("valuation.details")}</a>
    {/if}
  </aside>
{/if}
