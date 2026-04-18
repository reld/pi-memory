import type { MemoryRow } from "./memory.ts";

export interface RecallMemoryRow extends MemoryRow {
  recallScore: number;
}

export interface RecallMemoriesResult {
  items: RecallMemoryRow[];
}
