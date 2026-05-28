<script lang="ts">
  import _ from "lodash";

  export let title: string;
  export let value: string;
  export let small: boolean = false;
  export let narrow: boolean = false;
  export let color: string = null;
  export let subtitle: string = null;
</script>

<div class="level-item {narrow && 'is-narrow'} has-text-left" class:small>
  <div>
    <p class="heading">{title}</p>
    {#if color}
      <p class="title" style="padding: 5px; color: {color};">{value}</p>
    {:else}
      <p class="title has-text-grey-dark">{value}</p>
    {/if}
    {#if !_.isEmpty(subtitle)}
      <div class="sub-title">{@html subtitle}</div>
    {/if}
  </div>
</div>

<style lang="scss">
  @import "bulma/sass/utilities/_all.sass";

  // i64-R1: Bulma's .level-item is a flex item with the default min-width: auto
  // which can force it wider than the column when the inner text is long
  // (e.g. ¥846,251.66 under zh-CN). Allow it to shrink — the .title below
  // keeps the value on one line either way.
  .level-item {
    min-width: 0;
  }

  .level-item.small {
    .title {
      font-size: 1.25rem !important;
      line-height: 1.5rem !important;
    }
  }

  .heading {
    font-weight: 400 !important;
    font-size: 1rem !important;
    text-transform: capitalize !important;
    letter-spacing: normal !important;
    margin-bottom: 0 !important;
  }

  .title {
    font-weight: 800 !important;
    font-size: 1.75rem !important;
    line-height: 2rem !important;
    padding-left: 0 !important;
    padding-bottom: 0 !important;
    margin-bottom: 0 !important;
    // i64-R1: prevent ¥+thousands-grouped amounts (e.g. ¥846,251.66) from
    // wrapping mid-number when the LevelItem cell is narrower than the value.
    // The .level-item rule above also drops min-width: auto so flex can shrink
    // past the value width when truly necessary; in practice the parent .level
    // gives enough room and this keeps everything on one line.
    white-space: nowrap;
  }

  @include widescreen {
    .title {
      font-size: 2.25rem !important;
      line-height: 2.5rem !important;
    }

    .level-item.small {
      .title {
        font-size: 1.5rem !important;
        line-height: 1.75rem !important;
      }
    }
  }

  .sub-title {
    font-weight: normal !important;
    font-size: 0.75rem !important;
  }
</style>
