import { resolveRuntimeBehaviorConfig } from "./config.ts";

export interface ResolvedProjectContext {
  projectPath: string;
  storageBaseDir: string;
}

export function resolveStorageBaseDir(): string {
  return resolveRuntimeBehaviorConfig().storageBaseDir;
}

export function resolveProjectContext(cwd: string): ResolvedProjectContext {
  return {
    projectPath: cwd,
    storageBaseDir: resolveStorageBaseDir(),
  };
}
