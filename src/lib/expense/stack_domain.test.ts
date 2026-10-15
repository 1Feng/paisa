import { describe, expect, test } from "bun:test";
import { scaleLinear, stack, stackOffsetDiverging } from "d3";
import { expenseStackDomain } from "./stack_domain";

function series(points: Record<string, number>[], keys = ["dining", "insurance"]) {
  return stack().offset(stackOffsetDiverging).keys(keys)(points);
}

describe("expense chart domain", () => {
  test("refund month and positive months share zero without clipping either stack", () => {
    const data = series([
      { dining: 1000, insurance: 200 },
      { dining: 500, insurance: -14535.81 }
    ]);
    const domain = expenseStackDomain(data);
    expect(domain).toEqual([-14535.81, 1200]);
    const y = scaleLinear().domain(domain).range([300, 0]);
    for (const group of data) {
      for (const [bottom, top] of group) {
        expect(y(top)).toBeGreaterThanOrEqual(0);
        expect(y(bottom)).toBeLessThanOrEqual(300);
      }
    }
    expect(y(0)).toBeGreaterThan(0);
    expect(y(0)).toBeLessThan(300);
  });
  test("all-refund range retains a zero upper bound", () => {
    expect(expenseStackDomain(series([{ dining: -100, insurance: -200 }]))).toEqual([-300, 0]);
  });
  test("selecting a positive category removes the refund from the domain", () => {
    expect(expenseStackDomain(series([{ dining: 500, insurance: -14535.81 }], ["dining"]))).toEqual(
      [0, 500]
    );
  });
  test("empty and zero ranges have a usable scale", () => {
    expect(expenseStackDomain(series([]))).toEqual([0, 1]);
    expect(expenseStackDomain(series([{ dining: 0, insurance: 0 }]))).toEqual([0, 1]);
  });
});
