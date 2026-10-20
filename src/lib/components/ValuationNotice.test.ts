import { afterEach, describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { compile, preprocess } from "svelte/compiler";
import { tick } from "svelte";
import ts from "typescript";
import dayjs from "dayjs";
import * as i18n from "../i18n";
import type { ValuationCoverage } from "../utils";

// Compile and mount the real component; replace only its HTTP boundary.
// Svelte's internal lifecycle exports select the browser runtime in Bun,
// whose default `svelte` export otherwise uses the SSR (no-op onMount) runtime.
const source = readFileSync(`${import.meta.dir}/ValuationNotice.svelte`, "utf8");
const prepared = await preprocess(source, {
  script: ({ content }) => ({
    code: ts.transpileModule(content, {
      compilerOptions: { verbatimModuleSyntax: true, target: ts.ScriptTarget.ES2022 }
    }).outputText
  })
});
const compiled = compile(prepared.code, { generate: "dom" });
const code = ts.transpileModule(compiled.js.code, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText;
const nativeRequire = createRequire(import.meta.url);
let response: ValuationCoverage | Error;
const module = { exports: {} as { default: any } };
new Function("require", "module", "exports", code)(
  (name: string) => {
    if (name === "$lib/utils") {
      return {
        ajax: async () => {
          if (response instanceof Error) throw response;
          return response;
        }
      };
    }
    if (name === "$lib/i18n") return i18n;
    return nativeRequire(name === "svelte" ? "svelte/internal" : name);
  },
  module,
  module.exports
);
let component: { $destroy(): void };
async function render(coverage: ValuationCoverage | Error, locale = "en") {
  response = coverage;
  i18n.initI18n(locale);
  await i18n.waitLocale();
  component = new module.exports.default({ target: document.body });
  await new Promise((resolve) => setTimeout(resolve, 0));
  await tick();
  return document.body;
}
const complete: ValuationCoverage = {
  has_conversions: true,
  current_complete: true,
  current_missing: [],
  gaps: []
};
afterEach(() => {
  component?.$destroy();
  document.body.innerHTML = "";
});

describe("valuation coverage notice", () => {
  test("successful currency conversions do not warn", async () => {
    const body = await render(complete);
    expect(body.querySelectorAll("aside").length).toBe(0);
  });

  test("complete native-currency coverage does not warn", async () => {
    const body = await render({ ...complete, has_conversions: false });
    expect(body.querySelectorAll("aside").length).toBe(0);
  });

  test("historical gaps warn even without current conversions and preserve dates", async () => {
    const body = await render({
      ...complete,
      has_conversions: false,
      gaps: [
        {
          commodity: "TEST",
          currency: "USD",
          date: dayjs("2020-01-02"),
          until_date: dayjs("2020-02-03")
        }
      ]
    });
    expect(body.querySelector("aside")).not.toBeNull();
    expect(body.querySelector("li")?.textContent).toContain("TEST: 2020-01-02");
    expect(body.querySelector("li")?.textContent).toContain("2020-02-03");
    expect(body.querySelector("li")?.textContent).toContain("USD");
    expect(body.querySelector("li")?.textContent).toContain("end date excluded");
    expect(body.querySelector("a")?.getAttribute("href")).toBe("/ledger/price");
  });

  test("missing current quotes retain their commodity names and a stable cost boundary", async () => {
    const body = await render({
      ...complete,
      current_complete: false,
      current_missing: ["TEST"]
    });
    expect(body.textContent).toContain("TEST");
    expect(body.textContent).toContain(
      "Reference valuations do not validate historical acquisition costs or returns."
    );
    expect(body.textContent).not.toContain("separate task");
  });

  test("Chinese warning explains the same product boundary", async () => {
    const body = await render(
      { ...complete, current_complete: false, current_missing: ["TEST"] },
      "zh-CN"
    );
    expect(body.textContent).toContain("参考估值不代表历史购入成本或收益已核验。");
    expect(body.textContent).not.toContain("另案处理");
  });

  test("coverage request failures remain visible", async () => {
    const body = await render(new Error("unavailable"));
    expect(body.querySelector("aside")).not.toBeNull();
    expect(body.textContent).toContain("Price completeness could not be checked");
  });
});
