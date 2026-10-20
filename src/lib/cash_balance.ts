export type CashRule = {
  account: string;
  currency: string;
  kind: "cash" | "mixed" | "excluded";
  location: string;
  access: string;
  effective_from: string;
  effective_until?: string;
  evidence: string;
  verified_on?: string;
};
export type CashBalanceRow = {
  account: string;
  currency: string;
  native_amount: string;
  reference_amount: string | null;
  quote_date?: string;
  kind: "cash" | "mixed" | "excluded" | "unclassified";
  location: string;
  access: string;
  reason: string;
  last_posting?: string;
  rule?: CashRule;
};
export type CashBalance = {
  as_of: string;
  currency: string;
  reference_total: string;
  fx_complete: boolean;
  classification_complete: boolean;
  policy_configured: boolean;
  last_posting?: string;
  last_journal_sync: string | null;
  groups: Record<string, { reference_total: string; fx_complete: boolean }>;
  rows: CashBalanceRow[];
};

// This additive endpoint preserves decimal strings and plain calendar dates;
// do not send it through legacy money/date coercion.
export async function fetchCashBalance(date: string): Promise<CashBalance> {
  const response = await fetch(`/api/cash_balance?date=${encodeURIComponent(date)}`);
  if (!response.ok) throw new Error("Cash balances could not be loaded");
  return response.json();
}
