// Helpers for thinning out d3 axis tick labels when the underlying
// data set is dense (e.g. 8+ years of monthly samples on a band scale).
//
// Kept in its own module — without SvelteKit / d3 imports — so that
// `bun:test` can exercise the pure index math without bootstrapping
// the rest of `./utils`.

/**
 * Pick at most `maxTicks` indexes from `[0, numPoints - 1]`, evenly
 * spaced and always including the first and last index.
 *
 * Used to compute a subset of months / years to display as tick labels
 * on a `d3.scaleBand` axis when the raw data set is denser than the
 * X-axis can render legibly.
 */
export function evenlySpacedTickIndexes(numPoints: number, maxTicks: number): number[] {
  if (numPoints <= 0 || maxTicks <= 0) {
    return [];
  }
  if (numPoints <= maxTicks) {
    return Array.from({ length: numPoints }, (_v, i) => i);
  }
  if (maxTicks === 1) {
    return [0];
  }

  // step >= 1; spread `maxTicks` points across `numPoints - 1` slots.
  const step = (numPoints - 1) / (maxTicks - 1);
  const seen = new Set<number>();
  const result: number[] = [];
  for (let i = 0; i < maxTicks; i++) {
    const idx = Math.round(i * step);
    if (!seen.has(idx)) {
      seen.add(idx);
      result.push(idx);
    }
  }
  return result;
}

/**
 * Convenience: given a domain array (e.g. month labels for a
 * `d3.scaleBand`), return the subset of values that should receive a
 * tick label. Preserves original order and ensures the first and last
 * domain entries are always shown.
 */
export function evenlySpacedTickValues<T>(domain: readonly T[], maxTicks: number): T[] {
  return evenlySpacedTickIndexes(domain.length, maxTicks).map((i) => domain[i]);
}
