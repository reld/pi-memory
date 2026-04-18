import type { IngestSessionsResult } from "./ingest.ts";

export interface RebuildProjectMemoryResult {
  clearedMemorySources: number;
  clearedMemoryItems: number;
  clearedIngestionState: number;
  clearedIngestionRuns: number;
  ingest: IngestSessionsResult;
}
