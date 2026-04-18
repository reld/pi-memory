export interface IngestSessionsResult {
  runId: string;
  trackedSessionsDiscovered: number;
  sessionFilesProcessed: number;
  entriesSeen: number;
  candidatesFound: number;
  memoriesCreated: number;
  memoriesUpdated: number;
  memoriesIgnored: number;
  lastIngestedAt: string;
}
