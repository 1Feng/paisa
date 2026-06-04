import { afterEach, describe, expect, test } from "bun:test";
import { chartWidth } from "./chart_width";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("chartWidth", () => {
  test("returns max(parentWidth, fallback) when element has a parent", () => {
    const parent = document.createElement("div");
    const el = document.createElement("div");
    parent.appendChild(el);
    el.id = "chart";
    document.body.appendChild(parent);
    Object.defineProperty(parent, "clientWidth", { value: 1234, configurable: true });

    // parent wider than fallback -> parent width wins
    expect(chartWidth("chart", 800)).toBe(1234);
    // fallback wider than parent -> fallback wins
    expect(chartWidth("chart", 2000)).toBe(2000);
    // passing the element directly works too
    expect(chartWidth(el, 800)).toBe(1234);
  });

  test("returns fallback when the id is not found", () => {
    expect(chartWidth("does-not-exist", 850)).toBe(850);
    expect(chartWidth(null, 850)).toBe(850);
  });

  test("returns fallback when the element has no parent", () => {
    const orphan = document.createElement("div");
    // not attached to the DOM, so parentElement is null
    expect(orphan.parentElement).toBeNull();
    expect(chartWidth(orphan, 999)).toBe(999);
  });
});
