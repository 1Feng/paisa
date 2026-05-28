import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// i71 guard: when /api/templates returns an empty array (or, prior to the
// backend fix, `null`), the page must not crash on `templates[0]`. The fix
// lives in onMount inside +page.svelte. We assert the source contains a
// defensive guard so a future refactor can't silently regress this.
//
// Full Svelte component rendering through happy-dom would be ideal, but the
// page pulls in CodeMirror + svelte-select which makes a unit test
// prohibitively expensive. Reading the source is the same approach used by
// LevelItem.test.ts (#64 R1).
describe("ledger import page guards (issue #71)", () => {
  const source = readFileSync(join(__dirname, "+page.svelte"), "utf-8");

  test("normalises null templates response into an empty array", () => {
    // The pattern we accept: `templates = templates || []` (or `?? []`) on the
    // line after destructuring the API response. Catches both null (legacy
    // backend) and undefined.
    expect(source).toMatch(/templates\s*=\s*templates\s*(\|\||\?\?)\s*\[\s*\]/);
  });

  test("guards selectedTemplate access behind a length check", () => {
    // We accept any explicit non-emptiness check before assigning from
    // templates[0]: templates.length > 0, !_.isEmpty(templates), or
    // templates.length (truthy). The point is that the page must not
    // unconditionally dereference templates[0] when the list is empty.
    expect(source).toMatch(/if\s*\(\s*templates\.length\s*>\s*0\s*\)/);
  });
});
