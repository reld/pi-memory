import { homedir } from "node:os";
import { join } from "node:path";

export interface RuntimeBehaviorConfig {
  storageBaseDir: string;
  backendPathOverride?: string;
  sessionDirOverride?: string;
  debug: boolean;
  autoIngest: boolean;
  autoRecall: boolean;
  recallLimit: number;
  rawSessionSearchEnabled: boolean;
}

export function resolveRuntimeBehaviorConfig(): RuntimeBehaviorConfig {
  return {
    storageBaseDir: resolvePathEnv(process.env.PI_MEMORY_STORAGE_BASE_DIR) || join(homedir(), ".pi-memory"),
    backendPathOverride: resolvePathEnv(process.env.PI_MEMORY_BACKEND_PATH),
    sessionDirOverride: resolvePathEnv(process.env.PI_MEMORY_SESSION_DIR),
    debug: parseBooleanEnv(process.env.PI_MEMORY_DEBUG, false),
    autoIngest: parseBooleanEnv(process.env.PI_MEMORY_AUTO_INGEST, true),
    autoRecall: parseBooleanEnv(process.env.PI_MEMORY_AUTO_RECALL, true),
    recallLimit: parseNumberEnv(process.env.PI_MEMORY_RECALL_LIMIT, 5),
    rawSessionSearchEnabled: parseBooleanEnv(process.env.PI_MEMORY_RAW_SESSION_SEARCH_ENABLED, true),
  };
}

function resolvePathEnv(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

function parseBooleanEnv(value: string | undefined, defaultValue: boolean): boolean {
  if (value == null || value.trim() === "") {
    return defaultValue;
  }
  const normalized = value.trim().toLowerCase();
  return !(normalized === "0" || normalized === "false" || normalized === "no" || normalized === "off");
}

function parseNumberEnv(value: string | undefined, defaultValue: number): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return defaultValue;
  }
  return parsed;
}
