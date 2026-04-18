export interface MemoryRow {
  memoryId: string;
  category: string;
  summary: string;
  details?: string;
  status: string;
  confidence: number;
  importance: number;
  updatedAt: string;
  score?: number;
}

export interface ListMemoriesResult {
  items: MemoryRow[];
}

export interface SearchMemoriesResult {
  items: MemoryRow[];
}
