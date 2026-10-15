import type { Series } from "d3";

// Diverging stacks need separate positive and negative bounds. A net total
// can hide either side and leaves refunds outside a zero-based plot.
export function expenseStackDomain(
  series: Series<Record<string, number>, string>[]
): [number, number] {
  let min = 0;
  let max = 0;
  for (const group of series) {
    for (const [bottom, top] of group) {
      min = Math.min(min, bottom);
      max = Math.max(max, top);
    }
  }
  return min === max ? [0, 1] : [min, max];
}
