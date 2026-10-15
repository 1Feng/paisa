import fs from "fs";
import { diffString } from "json-diff";

export function verifySnapshot(filename: string, data: unknown, regenerate: boolean): void {
  if (regenerate) {
    fs.writeFileSync(filename, JSON.stringify(data, null, 2));
    return;
  }
  if (!fs.existsSync(filename)) {
    throw new Error(`Missing snapshot ${filename}; run make regen to generate it explicitly.`);
  }
  const current = JSON.parse(fs.readFileSync(filename, "utf8"));
  const diff = diffString(data, current, {
    excludeKeys: ["id", "transaction_id", "endLine", "transaction_end_line"]
  });
  if (diff !== "") throw new Error(diff);
}
