export interface ProjectInitResult {
  projectId: string;
  projectDir: string;
  projectFile: string;
  dbPath: string;
  created: boolean;
}

export interface ProjectMetadata {
  version: number;
  projectId: string;
  name: string;
  slug: string;
  hash: string;
  projectPath: string;
  projectRootStrategy: string;
  projectDir: string;
  dbPath: string;
  createdAt: string;
  updatedAt: string;
  lastOpenedAt?: string;
  status: string;
}

export interface GetProjectResult {
  initialized: boolean;
  project?: ProjectMetadata;
}

export interface ProjectStatusResult {
  initialized: boolean;
  projectId?: string;
  dbPath?: string;
  activeMemoryCount: number;
  trackedSessionCount: number;
  lastIngestedAt: string;
}
