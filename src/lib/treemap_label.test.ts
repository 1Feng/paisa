import { describe, expect, test } from "bun:test";
import { estimateTextWidth, fitTreemapLabel } from "./treemap_label";

describe("estimateTextWidth", () => {
  test("ASCII chars are estimated at roughly 0.6 em", () => {
    // 10 ASCII chars at 10px font ≈ 60px (0.6 em per char)
    expect(estimateTextWidth("abcdefghij", 10)).toBeCloseTo(60, 5);
  });

  test("CJK chars are estimated at roughly 1.0 em", () => {
    // 4 CJK chars at 10px font ≈ 40px
    expect(estimateTextWidth("活期存款", 10)).toBeCloseTo(40, 5);
  });

  test("empty string has zero width", () => {
    expect(estimateTextWidth("", 12)).toBe(0);
  });

  test("mixed ASCII + CJK sums correctly", () => {
    // "AB活期" => 2*0.6 + 2*1.0 = 3.2 em → 32 at 10px
    expect(estimateTextWidth("AB活期", 10)).toBeCloseTo(32, 5);
  });
});

describe("fitTreemapLabel", () => {
  const FONT = 12; // matches the default Bulma `.heading` size in pixels-ish

  test("returns the full label when the cell is wide enough", () => {
    // 'ASSETS' = 6 ASCII chars → ~43.2px at 12px font; cell 200px is ample
    expect(fitTreemapLabel("ASSETS", 200, FONT)).toBe("ASSETS");
  });

  test("returns truncated label with ellipsis when partial fit is possible", () => {
    // 'ASSETS:HOUSE:PRIMARY' at 12px font ≈ 144px; cell 60px should hold
    // ~7 chars (0.6*12*7 ≈ 50px) plus the ellipsis.
    const result = fitTreemapLabel("ASSETS:HOUSE:PRIMARY", 60, FONT);
    expect(result.endsWith("…")).toBe(true);
    expect(result.length).toBeLessThan("ASSETS:HOUSE:PRIMARY".length);
    expect(result.length).toBeGreaterThan(1);
    // The truncated prefix should be a real prefix of the original label
    const prefix = result.slice(0, -1);
    expect("ASSETS:HOUSE:PRIMARY".startsWith(prefix)).toBe(true);
  });

  test("returns empty string when even one ASCII char + ellipsis cannot fit", () => {
    // 12px font, cell width 10px: ellipsis alone ≈ 12*0.6 = 7.2px, budget
    // for prefix is 10 - 7.2 = 2.8px; one ASCII char = 7.2px → doesn't fit.
    expect(fitTreemapLabel("Anything", 10, FONT)).toBe("");
  });

  test("returns empty string when the cell is below the minimum renderable width", () => {
    // Below 16px nothing readable can fit no matter the label.
    expect(fitTreemapLabel("Hi", 5, FONT)).toBe("");
  });

  test("returns empty string when the label is empty", () => {
    expect(fitTreemapLabel("", 500, FONT)).toBe("");
  });

  test("CJK label that fits is returned verbatim", () => {
    // 4 chars * 1.0 em * 12px = 48px; cell 60px is enough.
    expect(fitTreemapLabel("活期存款", 60, FONT)).toBe("活期存款");
  });

  test("CJK label that doesn't fit is truncated with ellipsis", () => {
    const label = "活期存款理财基金";
    // Full width 8 * 12 = 96px; cell 40px (budget ≈ 32.8px) holds ~2 CJK chars
    const result = fitTreemapLabel(label, 40, FONT);
    expect(result.endsWith("…")).toBe(true);
    expect(label.startsWith(result.slice(0, -1))).toBe(true);
  });

  test("never splits a surrogate pair", () => {
    // An emoji (single codepoint, two UTF-16 units). Width estimated as
    // full-width. Cell wide enough only for the emoji + ellipsis: we should
    // get back the whole emoji or nothing, never a half-surrogate.
    const label = "\u{1F600}rest";
    const result = fitTreemapLabel(label, 30, FONT);
    // Result must not contain an unpaired surrogate code unit.
    for (let i = 0; i < result.length; i++) {
      const code = result.charCodeAt(i);
      if (code >= 0xd800 && code <= 0xdbff) {
        // high surrogate must be followed by low surrogate
        const next = result.charCodeAt(i + 1);
        expect(next).toBeGreaterThanOrEqual(0xdc00);
        expect(next).toBeLessThanOrEqual(0xdfff);
      }
    }
  });
});
