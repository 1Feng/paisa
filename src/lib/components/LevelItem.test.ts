import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// i64-R1 guard: large localized currency values (e.g. ¥846,251.66) overflow
// the Bulma .level-item card and wrap mid-number. The fix lives entirely in
// LevelItem.svelte's <style>; we read the source here so we don't need a full
// DOM testing harness. If someone removes the white-space: nowrap rule or the
// .level-item { min-width: 0 } override, this test fails loudly.
describe("LevelItem CSS guards (issue #64 R1)", () => {
  const source = readFileSync(join(__dirname, "LevelItem.svelte"), "utf-8");

  test(".title has white-space: nowrap so currency values stay on one line", () => {
    // Match the rule inside the .title block. Allow whitespace / comments
    // between lines, but require the declaration is present.
    expect(source).toMatch(/\.title\s*\{[\s\S]*?white-space:\s*nowrap;[\s\S]*?\}/);
  });

  test(".level-item has min-width: 0 so it can shrink within its column", () => {
    expect(source).toMatch(/\.level-item\s*\{[\s\S]*?min-width:\s*0;[\s\S]*?\}/);
  });
});
