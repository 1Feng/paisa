import { afterEach, beforeEach, describe, expect, test } from "bun:test";
import fs from "fs";
import os from "os";
import path from "path";
import { verifySnapshot } from "./snapshot";

let directory: string;
let filename: string;
beforeEach(() => {
  directory = fs.mkdtempSync(path.join(os.tmpdir(), "paisa-snapshot-"));
  filename = path.join(directory, "response.json");
});
afterEach(() => fs.rmSync(directory, { recursive: true, force: true }));

describe("regression snapshot writes", () => {
  test("normal verification preserves original bytes despite ignored ID changes", () => {
    const original = '{ "postings": [{ "id": 1, "amount": 123.45 }] }\n';
    fs.writeFileSync(filename, original);
    fs.utimesSync(filename, new Date(0), new Date(0));
    const modifiedAt = fs.statSync(filename).mtimeMs;
    verifySnapshot(filename, { postings: [{ id: 99, amount: 123.45 }] }, false);
    expect(fs.readFileSync(filename, "utf8")).toBe(original);
    expect(fs.statSync(filename).mtimeMs).toBe(modifiedAt);
  });
  test("normal verification reports semantic differences without overwriting", () => {
    const original = '{"amount": 123.45}\n';
    fs.writeFileSync(filename, original);
    expect(() => verifySnapshot(filename, { amount: 99 }, false)).toThrow();
    expect(fs.readFileSync(filename, "utf8")).toBe(original);
  });
  test("normal verification fails on missing snapshots instead of creating them", () => {
    expect(() => verifySnapshot(filename, { amount: 123.45 }, false)).toThrow("make regen");
    expect(fs.existsSync(filename)).toBe(false);
  });
  test("explicit regeneration overwrites an existing snapshot", () => {
    fs.writeFileSync(filename, '{"amount": 1}');
    verifySnapshot(filename, { amount: 123.45 }, true);
    expect(JSON.parse(fs.readFileSync(filename, "utf8"))).toEqual({ amount: 123.45 });
  });
  test("explicit regeneration creates a missing snapshot", () => {
    verifySnapshot(filename, { amount: 123.45 }, true);
    expect(JSON.parse(fs.readFileSync(filename, "utf8"))).toEqual({ amount: 123.45 });
  });
});
