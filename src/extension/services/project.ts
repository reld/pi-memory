import { homedir } from "node:os";
import { join } from "node:path";

export interface ResolvedProjectContext {
  projectPath: string;
  storageBaseDir: string;
}

export function resolveStorageBaseDir(): string {
  return process.env.PI_MEMORY_STORAGE_BASE_DIR?.trim() || join(homedir(), ".pi-memory");
}

export function resolveProjectContext(cwd: string): ResolvedProjectContext {
  return {
    projectPath: cwd,
    storageBaseDir: resolveStorageBaseDir(),
  };
}
