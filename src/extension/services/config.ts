import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

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

export type RuntimeBehaviorConfigKey = keyof RuntimeBehaviorConfig;
export type RuntimeBehaviorConfigSource = "default" | "file" | "env";

export interface RuntimeBehaviorConfigDetails {
  config: RuntimeBehaviorConfig;
  sources: Record<RuntimeBehaviorConfigKey, RuntimeBehaviorConfigSource>;
  storedConfig: Partial<RuntimeBehaviorConfig>;
  configFilePath: string;
}

const DEFAULT_STORAGE_BASE_DIR = join(homedir(), ".pi-memory");
const CONFIG_FILE_PATH = join(homedir(), ".config", "pi-memory", "config.json");

const CONFIG_KEY_ALIASES: Record<string, RuntimeBehaviorConfigKey> = {
  storageBaseDir: "storageBaseDir",
  "storage-base-dir": "storageBaseDir",
  backendPathOverride: "backendPathOverride",
  "backend-path": "backendPathOverride",
  sessionDirOverride: "sessionDirOverride",
  "session-dir": "sessionDirOverride",
  debug: "debug",
  autoIngest: "autoIngest",
  "auto-ingest": "autoIngest",
  autoRecall: "autoRecall",
  "auto-recall": "autoRecall",
  recallLimit: "recallLimit",
  "recall-limit": "recallLimit",
  rawSessionSearchEnabled: "rawSessionSearchEnabled",
  "raw-session-search": "rawSessionSearchEnabled",
};

export function resolveRuntimeBehaviorConfig(): RuntimeBehaviorConfig {
  return resolveRuntimeBehaviorConfigDetails().config;
}

export function resolveRuntimeBehaviorConfigDetails(): RuntimeBehaviorConfigDetails {
  const storedConfig = readStoredConfigFile();

  const storageBaseDir = resolvePathValue(
    process.env.PI_MEMORY_STORAGE_BASE_DIR,
    storedConfig.storageBaseDir,
    DEFAULT_STORAGE_BASE_DIR,
  );
  const backendPathOverride = resolveOptionalPathValue(process.env.PI_MEMORY_BACKEND_PATH, storedConfig.backendPathOverride);
  const sessionDirOverride = resolveOptionalPathValue(process.env.PI_MEMORY_SESSION_DIR, storedConfig.sessionDirOverride);
  const debug = resolveBooleanValue(process.env.PI_MEMORY_DEBUG, storedConfig.debug, false);
  const autoIngest = resolveBooleanValue(process.env.PI_MEMORY_AUTO_INGEST, storedConfig.autoIngest, true);
  const autoRecall = resolveBooleanValue(process.env.PI_MEMORY_AUTO_RECALL, storedConfig.autoRecall, true);
  const recallLimit = resolveNumberValue(process.env.PI_MEMORY_RECALL_LIMIT, storedConfig.recallLimit, 5);
  const rawSessionSearchEnabled = resolveBooleanValue(
    process.env.PI_MEMORY_RAW_SESSION_SEARCH_ENABLED,
    storedConfig.rawSessionSearchEnabled,
    true,
  );

  return {
    config: {
      storageBaseDir: storageBaseDir.value,
      backendPathOverride: backendPathOverride.value,
      sessionDirOverride: sessionDirOverride.value,
      debug: debug.value,
      autoIngest: autoIngest.value,
      autoRecall: autoRecall.value,
      recallLimit: recallLimit.value,
      rawSessionSearchEnabled: rawSessionSearchEnabled.value,
    },
    sources: {
      storageBaseDir: storageBaseDir.source,
      backendPathOverride: backendPathOverride.source,
      sessionDirOverride: sessionDirOverride.source,
      debug: debug.source,
      autoIngest: autoIngest.source,
      autoRecall: autoRecall.source,
      recallLimit: recallLimit.source,
      rawSessionSearchEnabled: rawSessionSearchEnabled.source,
    },
    storedConfig,
    configFilePath: CONFIG_FILE_PATH,
  };
}

export function normalizeRuntimeBehaviorConfigKey(input: string): RuntimeBehaviorConfigKey | undefined {
  return CONFIG_KEY_ALIASES[input.trim()];
}

export function listRuntimeBehaviorConfigKeys(): RuntimeBehaviorConfigKey[] {
  return [
    "storageBaseDir",
    "backendPathOverride",
    "sessionDirOverride",
    "debug",
    "autoIngest",
    "autoRecall",
    "recallLimit",
    "rawSessionSearchEnabled",
  ];
}

export function setStoredRuntimeBehaviorConfigValue(key: RuntimeBehaviorConfigKey, rawValue: string): RuntimeBehaviorConfig {
  const storedConfig = readStoredConfigFile();
  const nextConfig = {
    ...storedConfig,
    [key]: parseStoredConfigValue(key, rawValue),
  } as Partial<RuntimeBehaviorConfig>;
  writeStoredConfigFile(nextConfig);
  return resolveRuntimeBehaviorConfig();
}

export function unsetStoredRuntimeBehaviorConfigValue(key: RuntimeBehaviorConfigKey): RuntimeBehaviorConfig {
  const storedConfig = readStoredConfigFile();
  const nextConfig = { ...storedConfig } as Partial<RuntimeBehaviorConfig>;
  delete nextConfig[key];
  writeStoredConfigFile(nextConfig);
  return resolveRuntimeBehaviorConfig();
}

function readStoredConfigFile(): Partial<RuntimeBehaviorConfig> {
  if (!existsSync(CONFIG_FILE_PATH)) {
    return {};
  }

  try {
    const parsed = JSON.parse(readFileSync(CONFIG_FILE_PATH, "utf8")) as Record<string, unknown>;
    return sanitizeStoredConfig(parsed);
  } catch {
    return {};
  }
}

function writeStoredConfigFile(config: Partial<RuntimeBehaviorConfig>): void {
  mkdirSync(dirname(CONFIG_FILE_PATH), { recursive: true });
  writeFileSync(CONFIG_FILE_PATH, `${JSON.stringify(sortStoredConfig(config), null, 2)}\n`, "utf8");
}

function sortStoredConfig(config: Partial<RuntimeBehaviorConfig>): Partial<RuntimeBehaviorConfig> {
  return listRuntimeBehaviorConfigKeys().reduce<Partial<RuntimeBehaviorConfig>>((sorted, key) => {
    const value = config[key];
    if (value !== undefined) {
      return {
        ...sorted,
        [key]: value,
      } as Partial<RuntimeBehaviorConfig>;
    }
    return sorted;
  }, {});
}

function sanitizeStoredConfig(parsed: Record<string, unknown>): Partial<RuntimeBehaviorConfig> {
  const sanitized: Partial<RuntimeBehaviorConfig> = {};

  const storageBaseDir = trimString(parsed.storageBaseDir);
  if (storageBaseDir) {
    sanitized.storageBaseDir = storageBaseDir;
  }

  const backendPathOverride = trimString(parsed.backendPathOverride);
  if (backendPathOverride) {
    sanitized.backendPathOverride = backendPathOverride;
  }

  const sessionDirOverride = trimString(parsed.sessionDirOverride);
  if (sessionDirOverride) {
    sanitized.sessionDirOverride = sessionDirOverride;
  }

  if (typeof parsed.debug === "boolean") {
    sanitized.debug = parsed.debug;
  }
  if (typeof parsed.autoIngest === "boolean") {
    sanitized.autoIngest = parsed.autoIngest;
  }
  if (typeof parsed.autoRecall === "boolean") {
    sanitized.autoRecall = parsed.autoRecall;
  }
  if (typeof parsed.rawSessionSearchEnabled === "boolean") {
    sanitized.rawSessionSearchEnabled = parsed.rawSessionSearchEnabled;
  }
  if (typeof parsed.recallLimit === "number" && Number.isFinite(parsed.recallLimit) && parsed.recallLimit > 0) {
    sanitized.recallLimit = parsed.recallLimit;
  }

  return sanitized;
}

function parseStoredConfigValue(key: RuntimeBehaviorConfigKey, rawValue: string): RuntimeBehaviorConfig[RuntimeBehaviorConfigKey] {
  const trimmed = rawValue.trim();

  switch (key) {
    case "storageBaseDir":
    case "backendPathOverride":
    case "sessionDirOverride": {
      if (!trimmed) {
        throw new Error(`Config value for ${key} must not be empty.`);
      }
      return trimmed;
    }
    case "debug":
    case "autoIngest":
    case "autoRecall":
    case "rawSessionSearchEnabled":
      return parseBooleanStrict(trimmed);
    case "recallLimit": {
      const parsed = Number(trimmed);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        throw new Error("Config value for recallLimit must be a positive number.");
      }
      return parsed;
    }
  }
}

function resolvePathValue(envValue: string | undefined, fileValue: string | undefined, defaultValue: string) {
  const envResolved = trimString(envValue);
  if (envResolved) {
    return { value: envResolved, source: "env" as const };
  }
  if (fileValue) {
    return { value: fileValue, source: "file" as const };
  }
  return { value: defaultValue, source: "default" as const };
}

function resolveOptionalPathValue(envValue: string | undefined, fileValue: string | undefined) {
  const envResolved = trimString(envValue);
  if (envResolved) {
    return { value: envResolved, source: "env" as const };
  }
  if (fileValue) {
    return { value: fileValue, source: "file" as const };
  }
  return { value: undefined, source: "default" as const };
}

function resolveBooleanValue(envValue: string | undefined, fileValue: boolean | undefined, defaultValue: boolean) {
  if (envValue != null && envValue.trim() !== "") {
    return { value: parseBooleanLoose(envValue, defaultValue), source: "env" as const };
  }
  if (typeof fileValue === "boolean") {
    return { value: fileValue, source: "file" as const };
  }
  return { value: defaultValue, source: "default" as const };
}

function resolveNumberValue(envValue: string | undefined, fileValue: number | undefined, defaultValue: number) {
  if (envValue != null && envValue.trim() !== "") {
    return { value: parseNumberLoose(envValue, defaultValue), source: "env" as const };
  }
  if (typeof fileValue === "number" && Number.isFinite(fileValue) && fileValue > 0) {
    return { value: fileValue, source: "file" as const };
  }
  return { value: defaultValue, source: "default" as const };
}

function trimString(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function parseBooleanLoose(value: string, defaultValue: boolean): boolean {
  const trimmed = value.trim();
  if (!trimmed) {
    return defaultValue;
  }
  const normalized = trimmed.toLowerCase();
  return !(normalized === "0" || normalized === "false" || normalized === "no" || normalized === "off");
}

function parseBooleanStrict(value: string): boolean {
  const normalized = value.trim().toLowerCase();
  if (["1", "true", "yes", "on"].includes(normalized)) {
    return true;
  }
  if (["0", "false", "no", "off"].includes(normalized)) {
    return false;
  }
  throw new Error("Config boolean values must be one of: true/false, on/off, yes/no, 1/0.");
}

function parseNumberLoose(value: string, defaultValue: number): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return defaultValue;
  }
  return parsed;
}
