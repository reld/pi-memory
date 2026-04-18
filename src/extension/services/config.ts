export interface RuntimeBehaviorConfig {
  autoIngest: boolean;
  autoRecall: boolean;
  recallLimit: number;
}

export function resolveRuntimeBehaviorConfig(): RuntimeBehaviorConfig {
  return {
    autoIngest: parseBooleanEnv(process.env.PI_MEMORY_AUTO_INGEST, true),
    autoRecall: parseBooleanEnv(process.env.PI_MEMORY_AUTO_RECALL, true),
    recallLimit: parseNumberEnv(process.env.PI_MEMORY_RECALL_LIMIT, 5),
  };
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
