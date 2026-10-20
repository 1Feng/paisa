import { afterEach, expect, test } from "bun:test";
import { fetchCashBalance } from "./cash_balance";
const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
});
test("cash API preserves precise money strings and does not turn missing FX into zero", async () => {
  let url = "";
  globalThis.fetch = (async (input) => {
    url = String(input);
    return new Response(
      JSON.stringify({
        as_of: "2024-01-02",
        reference_total: "9007199254740993.1234",
        rows: [{ reference_amount: null, native_amount: "0.00000001" }]
      })
    );
  }) as typeof fetch;
  const data = await fetchCashBalance("2024-01-02");
  expect(url).toBe("/api/cash_balance?date=2024-01-02");
  expect(data.as_of).toBe("2024-01-02");
  expect(data.reference_total).toBe("9007199254740993.1234");
  expect(data.rows[0].reference_amount).toBeNull();
});
test("failed cash API rejects instead of returning an empty zero balance", async () => {
  globalThis.fetch = (async () => new Response("failed", { status: 500 })) as typeof fetch;
  await expect(fetchCashBalance("2024-01-02")).rejects.toThrow();
});
