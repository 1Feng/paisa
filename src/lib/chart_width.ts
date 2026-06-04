/**
 * Compute the available width for a d3 chart given its container element.
 * Returns at least `fallback` if the element is missing, detached, or has
 * no parent (e.g. during client-side route transitions or in test envs).
 */
export function chartWidth(idOrElement: string | Element | null, fallback: number): number {
  const el = typeof idOrElement === "string" ? document.getElementById(idOrElement) : idOrElement;
  const parent = el?.parentElement;
  return Math.max(parent?.clientWidth ?? fallback, fallback);
}
