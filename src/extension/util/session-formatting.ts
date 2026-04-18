import { basename } from "node:path";

import type { SessionSearchRow } from "../types/session-search.ts";

export function formatSessionSearchRows(title: string, items: SessionSearchRow[]): string {
  if (items.length === 0) {
    return `${title}\n- no session matches found`;
  }

  return [
    title,
    ...items.map((item) => {
      const filePart = basename(item.sessionFile);
      const rolePart = item.role ? ` ${item.role}` : "";
      const entryPart = item.entryId ? ` ${item.entryId}` : "";
      return `- [${filePart}${rolePart}${entryPart}] ${item.excerpt} (score: ${item.score.toFixed(2)})`;
    }),
  ].join("\n");
}
