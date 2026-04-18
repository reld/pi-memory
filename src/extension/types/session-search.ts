export interface SessionSearchRow {
  sessionFile: string;
  entryId?: string;
  role?: string;
  excerpt: string;
  score: number;
}

export interface SearchSessionsResult {
  items: SessionSearchRow[];
}
