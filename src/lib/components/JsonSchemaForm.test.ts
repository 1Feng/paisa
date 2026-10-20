import { afterEach, describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { compile, preprocess } from "svelte/compiler";
import { tick } from "svelte";
import ts from "typescript";

// Render the real recursive form. Only unused rich widgets are stubbed;
// svelte:self remains the compiled component, including all input bindings.
const source = readFileSync(`${import.meta.dir}/JsonSchemaForm.svelte`, "utf8");
const prepared = await preprocess(source, {
  script: ({ content }) => ({
    code: ts.transpileModule(content, {
      compilerOptions: { verbatimModuleSyntax: true, target: ts.ScriptTarget.ES2022 }
    }).outputText
  })
});
const compiled = compile(prepared.code, { generate: "dom", accessors: true });
const code = ts.transpileModule(compiled.js.code, {
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022,
    esModuleInterop: true
  }
}).outputText;
const nativeRequire = createRequire(import.meta.url);
const module = { exports: {} as { default: any } };
new Function("require", "module", "exports", code)(
  (name: string) => {
    if (name.endsWith(".svelte") || name === "svelte-select") return { default: class {} };
    if (name === "$lib/icon") return { iconsList: [], iconGlyph: () => "" };
    return nativeRequire(name === "svelte" ? "svelte/internal" : name);
  },
  module,
  module.exports
);
let component: any;
async function render(schema: any, value?: any, depth = 0) {
  component = new module.exports.default({
    target: document.body,
    props: { key: "items", schema, value, depth, allAccounts: [] }
  });
  await tick();
}
async function add() {
  document.querySelector<HTMLAnchorElement>(".config-add")!.click();
  await tick();
}
afterEach(() => {
  component?.$destroy();
  document.body.innerHTML = "";
});

describe("configuration array editing", () => {
  test.each([{ value: undefined }, { value: [] }])(
    "adds an editable string when the array is %j and has no default",
    async ({ value }) => {
      await render({ type: "array", items: { type: "string" } }, value);
      await add();
      expect(component.value).toEqual([""]);
      const input = document.querySelector<HTMLInputElement>("input")!;
      input.value = "Income:Employment:Wages";
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await tick();
      expect(component.value).toEqual(["Income:Employment:Wages"]);
    }
  );

  test("empty array defaults initialize only required object fields", async () => {
    await render({
      type: "array",
      default: [],
      items: {
        type: "object",
        required: ["account", "metadata"],
        properties: {
          account: { type: "string" },
          reviewed_on: { type: "string", format: "date" },
          metadata: {
            type: "object",
            required: ["active"],
            properties: { active: { type: "boolean" }, note: { type: "string" } }
          }
        }
      }
    });
    await add();
    expect(component.value).toEqual([{ account: "", metadata: { active: false } }]);
    // Opening the recursive object and editing must not create optional dates.
    document.querySelector<HTMLAnchorElement>(".config-body .config-header a")!.click();
    await tick();
    const input = document.querySelector<HTMLInputElement>("input")!;
    input.value = "Assets:Cash";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    await tick();
    expect(component.value[0].account).toBe("Assets:Cash");
    expect(Object.hasOwn(component.value[0], "reviewed_on")).toBe(false);
  });

  test("expanding an omitted array and editing a new object keeps optional evidence absent", async () => {
    await render(
      {
        type: "array",
        items: {
          type: "object",
          required: ["account", "currency", "kind"],
          properties: {
            account: { type: "string", minLength: 1 },
            currency: { type: "string", minLength: 1 },
            kind: { type: "string", enum: ["cash", "excluded"] },
            location: { type: "string", enum: ["bank", "other"] },
            verified_on: { type: "string", minLength: 1 },
            evidence: { type: "string", minLength: 1, "ui:widget": "textarea" }
          }
        }
      },
      undefined,
      1
    );
    document.querySelector<HTMLAnchorElement>(".config-header a")!.click();
    await tick();
    expect(document.querySelectorAll(".config-delete").length).toBe(0);
    await add();
    document.querySelector<HTMLAnchorElement>(".config-body .config-header a")!.click();
    await tick();
    const inputs = [...document.querySelectorAll<HTMLInputElement>("input")];
    for (const [index, value] of ["Assets:Cash", "CNY"].entries()) {
      inputs[index].value = value;
      inputs[index].dispatchEvent(new Event("input", { bubbles: true }));
    }
    await tick();
    expect(JSON.parse(JSON.stringify(component.value))).toEqual([
      { account: "Assets:Cash", currency: "CNY", kind: "cash" }
    ]);
    // Optional evidence can be supplied and removed again without persisting
    // empty strings rejected by minLength/date validation on save.
    const evidence = document.querySelector<HTMLTextAreaElement>("textarea")!;
    const date = inputs[2];
    for (const [field, text] of [
      [evidence, "Public statement"],
      [date, "2026-01-02"]
    ] as const) {
      field.value = text;
      field.dispatchEvent(new Event("input", { bubbles: true }));
    }
    await tick();
    expect(component.value[0].evidence).toBe("Public statement");
    expect(component.value[0].verified_on).toBe("2026-01-02");
    for (const field of [evidence, date]) {
      field.value = "";
      field.dispatchEvent(new Event("input", { bubbles: true }));
    }
    await tick();
    expect(JSON.parse(JSON.stringify(component.value))).toEqual([
      { account: "Assets:Cash", currency: "CNY", kind: "cash" }
    ]);
    const selects = [...document.querySelectorAll<HTMLSelectElement>("select")];
    selects[0].value = "excluded";
    selects[0].dispatchEvent(new Event("change", { bubbles: true }));
    selects[1].value = "bank";
    selects[1].dispatchEvent(new Event("change", { bubbles: true }));
    await tick();
    expect(component.value[0].kind).toBe("excluded");
    expect(component.value[0].location).toBe("bank");
    selects[1].selectedIndex = 0;
    selects[1].dispatchEvent(new Event("change", { bubbles: true }));
    await tick();
    expect(JSON.parse(JSON.stringify(component.value))).toEqual([
      { account: "Assets:Cash", currency: "CNY", kind: "excluded" }
    ]);
    inputs[0].value = "";
    inputs[0].dispatchEvent(new Event("input", { bubbles: true }));
    await tick();
    expect(component.value[0].account).toBe("");
  });

  test("legacy item defaults are deeply cloned and deletion preserves existing data", async () => {
    const template = { account: "Assets:Example", nested: { note: "template" } };
    const existing = { account: "Assets:Existing", nested: { note: "keep" } };
    await render(
      { type: "array", default: [template], items: { type: "object", properties: {} } },
      [existing]
    );
    await add();
    component.value[0].nested.note = "edited";
    expect(template.nested.note).toBe("template");
    expect(existing.nested.note).toBe("keep");
    await add();
    expect(component.value[0].nested.note).toBe("template");
    document.querySelector<HTMLAnchorElement>(".config-delete")!.click();
    await tick();
    expect(component.value).toEqual([
      { account: "Assets:Example", nested: { note: "edited" } },
      existing
    ]);
  });
});
