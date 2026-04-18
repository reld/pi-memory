import type { MemoryRow } from "../types/memory.ts";

export function formatMemoryRows(title: string, items: MemoryRow[]): string {
  if (items.length === 0) {
    return `${title}\n- no memories found`;
  }

  return [
    title,
    ...items.map((item) => {
      const scorePart = typeof item.score === "number" ? `, score: ${item.score.toFixed(2)}` : "";
      return `- [${item.memoryId}] ${item.category}: ${item.summary} (importance: ${item.importance.toFixed(2)}, confidence: ${item.confidence.toFixed(2)}${scorePart})`;
    }),
  ].join("\n");
}
