<script lang="ts">
  import CashBalances from "$lib/components/CashBalances.svelte";
  import AssetsBalance from "$lib/components/AssetsBalance.svelte";
  import { ajax, type AssetBreakdown } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";

  const cashEnabled = !!(
    USER_CONFIG.cash_accounts?.length || USER_CONFIG.cash_review_accounts?.length
  );
  let breakdowns: Record<string, AssetBreakdown> = {};

  onMount(async () => {
    ({ asset_breakdowns: breakdowns } = await ajax("/api/assets/balance"));
  });
</script>

<section class="section pb-0">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12 pb-0">
        {#if cashEnabled}
          <CashBalances />
        {/if}
        <AssetsBalance {breakdowns} />
      </div>
    </div>
  </div>
</section>
