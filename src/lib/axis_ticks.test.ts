import { describe, expect, test } from "bun:test";
import { evenlySpacedTickIndexes, evenlySpacedTickValues } from "./axis_ticks";

describe("evenlySpacedTickIndexes", () => {
  test("returns every index when numPoints <= maxTicks", () => {
    expect(evenlySpacedTickIndexes(5, 12)).toEqual([0, 1, 2, 3, 4]);
    expect(evenlySpacedTickIndexes(12, 12)).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11]);
  });

  test("returns empty when no data", () => {
    expect(evenlySpacedTickIndexes(0, 12)).toEqual([]);
    expect(evenlySpacedTickIndexes(10, 0)).toEqual([]);
  });

  test("limits dense data to at most maxTicks ticks", () => {
    // 8 years of monthly data = 96 points.
    const ticks = evenlySpacedTickIndexes(96, 12);
    expect(ticks.length).toBeLessThanOrEqual(12);
    expect(ticks.length).toBeGreaterThan(0);
  });

  test("always includes first and last index", () => {
    const ticks = evenlySpacedTickIndexes(96, 12);
    expect(ticks[0]).toBe(0);
    expect(ticks[ticks.length - 1]).toBe(95);
  });

  test("ticks are strictly increasing and in domain", () => {
    const ticks = evenlySpacedTickIndexes(96, 12);
    for (let i = 0; i < ticks.length; i++) {
      expect(ticks[i]).toBeGreaterThanOrEqual(0);
      expect(ticks[i]).toBeLessThan(96);
      if (i > 0) {
        expect(ticks[i]).toBeGreaterThan(ticks[i - 1]);
      }
    }
  });

  test("spacing is approximately uniform", () => {
    const ticks = evenlySpacedTickIndexes(96, 12);
    const gaps: number[] = [];
    for (let i = 1; i < ticks.length; i++) {
      gaps.push(ticks[i] - ticks[i - 1]);
    }
    const min = Math.min(...gaps);
    const max = Math.max(...gaps);
    // Allow at most a 1-step rounding variation between min and max gap.
    expect(max - min).toBeLessThanOrEqual(1);
  });

  test("handles maxTicks = 1 by returning the first index only", () => {
    expect(evenlySpacedTickIndexes(100, 1)).toEqual([0]);
  });

  test("dedupes when rounding collapses adjacent indexes", () => {
    // numPoints == maxTicks already returns all; pick a denser case
    // where rounding could repeat: 3 points into 5 ticks shouldn't
    // produce duplicates if numPoints<=maxTicks branch fires.
    expect(evenlySpacedTickIndexes(3, 5)).toEqual([0, 1, 2]);
  });
});

describe("evenlySpacedTickValues", () => {
  test("returns subset of original values in original order", () => {
    const months = Array.from({ length: 96 }, (_v, i) => `m${i}`);
    const subset = evenlySpacedTickValues(months, 12);
    expect(subset.length).toBeLessThanOrEqual(12);
    expect(subset[0]).toBe("m0");
    expect(subset[subset.length - 1]).toBe("m95");
    // Order preserved.
    const sorted = [...subset].sort((a, b) => Number(a.slice(1)) - Number(b.slice(1)));
    expect(subset).toEqual(sorted);
  });

  test("short domain passes through unchanged", () => {
    const months = ["jan", "feb", "mar"];
    expect(evenlySpacedTickValues(months, 12)).toEqual(["jan", "feb", "mar"]);
  });
});
