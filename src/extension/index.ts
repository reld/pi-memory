import { Type } from "@sinclair/typebox";

import { BackendError, forgetMemory, getProjectStatus, ingestSessions, initProject, listMemories, recallMemories, rebuildProjectMemory, rememberMemory, searchMemories, searchSessions } from "./services/backend.ts";
import { resolveRuntimeBehaviorConfig } from "./services/config.ts";
import { resolveProjectContext } from "./services/project.ts";
import { formatStatusBlock } from "./util/formatting.ts";
import { formatMemoryRows } from "./util/memory-formatting.ts";
import { formatSessionSearchRows } from "./util/session-formatting.ts";

const MEMORY_RECALL_PARAMS = Type.Object({
  limit: Type.Optional(Type.Number({ minimum: 1, maximum: 20 })),
});

const MEMORY_SEARCH_PARAMS = Type.Object({
  query: Type.String({ minLength: 1 }),
  limit: Type.Optional(Type.Number({ minimum: 1, maximum: 20 })),
});

const SESSION_SEARCH_PARAMS = Type.Object({
  query: Type.String({ minLength: 1 }),
  limit: Type.Optional(Type.Number({ minimum: 1, maximum: 20 })),
});

export default function createPiMemoryExtension(pi: any) {
  const runtimeConfig = resolveRuntimeBehaviorConfig();

  pi.on("session_start", async (_event: unknown, ctx: any) => {
    const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

    try {
      const status = await getProjectStatus({ projectPath, storageBaseDir });
      if (!status.initialized) {
        return;
      }

      if (runtimeConfig.autoIngest) {
        await ingestSessions({
          projectPath,
          storageBaseDir,
          trigger: "session_start_catchup",
          sessionDir: runtimeConfig.sessionDirOverride,
          activeSessionFile: ctx.sessionManager?.getSessionFile?.(),
        });
      }

      await notifyAutoRecall({
        ctx,
        projectPath,
        storageBaseDir,
        runtimeConfig,
      });
    } catch (error) {
      handleError(ctx, error, "Pi Memory session-start sync failed.");
    }
  });

  pi.on("turn_end", async (_event: unknown, ctx: any) => {
    if (!runtimeConfig.autoIngest) {
      return;
    }

    const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

    try {
      const status = await getProjectStatus({ projectPath, storageBaseDir });
      if (!status.initialized) {
        return;
      }

      await ingestSessions({
        projectPath,
        storageBaseDir,
        trigger: "auto_turn",
        sessionDir: runtimeConfig.sessionDirOverride,
        activeSessionFile: ctx.sessionManager?.getSessionFile?.(),
      });
    } catch (error) {
      handleError(ctx, error, "Pi Memory auto-ingest failed.");
    }
  });

  pi.registerTool({
    name: "pi_memory_recall",
    label: "Pi Memory Recall",
    description: "Recall the most relevant stored project memories.",
    promptSnippet: "Recall relevant stored project memories",
    promptGuidelines: [
      "Use this when the user asks what was discussed before, where work left off, or what should be remembered from prior sessions.",
    ],
    parameters: MEMORY_RECALL_PARAMS,
    async execute(_toolCallId: string, params: { limit?: number }, _signal: AbortSignal | undefined, _onUpdate: unknown, ctx: any) {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);
      await ensureProjectInitialized(projectPath, storageBaseDir);

      const result = await recallMemories({
        projectPath,
        storageBaseDir,
        limit: params.limit ?? runtimeConfig.recallLimit,
      });

      return {
        content: [{ type: "text", text: formatMemoryRows("Relevant project memory", result.items) }],
        details: { items: result.items },
      };
    },
  });

  pi.registerTool({
    name: "pi_memory_search",
    label: "Pi Memory Search",
    description: "Search structured stored project memories.",
    promptSnippet: "Search structured project memories",
    promptGuidelines: [
      "Use this for a specific remembered preference, decision, fact, task, or convention.",
    ],
    parameters: MEMORY_SEARCH_PARAMS,
    async execute(_toolCallId: string, params: { query: string; limit?: number }, _signal: AbortSignal | undefined, _onUpdate: unknown, ctx: any) {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);
      await ensureProjectInitialized(projectPath, storageBaseDir);

      const result = await searchMemories({
        projectPath,
        storageBaseDir,
        query: params.query.trim(),
        limit: params.limit ?? 10,
      });

      return {
        content: [{ type: "text", text: formatMemoryRows(`Pi Memory search: ${params.query.trim()}`, result.items) }],
        details: { items: result.items, query: params.query.trim() },
      };
    },
  });

  pi.registerTool({
    name: "pi_memory_search_sessions",
    label: "Pi Memory Session Search",
    description: "Search raw tracked Pi session history as a fallback.",
    promptSnippet: "Search raw tracked session history as a fallback",
    promptGuidelines: [
      "Use this only if structured memory is insufficient and the user asks about prior conversation details.",
    ],
    parameters: SESSION_SEARCH_PARAMS,
    async execute(_toolCallId: string, params: { query: string; limit?: number }, _signal: AbortSignal | undefined, _onUpdate: unknown, ctx: any) {
      if (!runtimeConfig.rawSessionSearchEnabled) {
        throw new Error("Raw session search is disabled by configuration.");
      }

      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);
      await ensureProjectInitialized(projectPath, storageBaseDir);

      const result = await searchSessions({
        projectPath,
        storageBaseDir,
        sessionDir: runtimeConfig.sessionDirOverride,
        query: params.query.trim(),
        limit: params.limit ?? 10,
      });

      return {
        content: [{ type: "text", text: formatSessionSearchRows(`Pi Memory raw session search: ${params.query.trim()}`, result.items) }],
        details: { items: result.items, query: params.query.trim() },
      };
    },
  });

  pi.registerCommand("pi-memory-init", {
    description: "Initialize Pi Memory for the current project",
    handler: async (_args: string, ctx: any) => {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await initProject({
          projectPath,
          storageBaseDir,
        });

        ctx.ui?.notify?.("Pi Memory initialized.", "info");
        ctx.ui?.notify?.(
          formatStatusBlock("Pi Memory initialized", [
            `project id: ${result.projectId}`,
            `project dir: ${result.projectDir}`,
            `db path: ${result.dbPath}`,
          ]),
          "info",
        );
      } catch (error) {
        handleError(ctx, error, "Failed to initialize Pi Memory.");
      }
    },
  });

  pi.registerCommand("pi-memory-status", {
    description: "Show Pi Memory status for the current project",
    handler: async (_args: string, ctx: any) => {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const status = await getProjectStatus({ projectPath, storageBaseDir });

        if (!status.initialized) {
          ctx.ui?.notify?.("Pi Memory is not initialized for this project yet.", "info");
          return;
        }

        ctx.ui?.notify?.(
          formatStatusBlock("Pi Memory status", [
            `project id: ${status.projectId ?? "unknown"}`,
            `db path: ${status.dbPath ?? "unknown"}`,
            `storage base dir: ${storageBaseDir}`,
            `session dir override: ${runtimeConfig.sessionDirOverride ?? "default"}`,
            `active memories: ${status.activeMemoryCount}`,
            `tracked sessions: ${status.trackedSessionCount}`,
            `last ingested at: ${status.lastIngestedAt || "never"}`,
            `auto ingest: ${runtimeConfig.autoIngest ? "on" : "off"}`,
            `auto recall: ${runtimeConfig.autoRecall ? "on" : "off"}`,
            `raw session search: ${runtimeConfig.rawSessionSearchEnabled ? "on" : "off"}`,
          ]),
          "info",
        );
      } catch (error) {
        handleError(ctx, error, "Failed to load Pi Memory status.");
      }
    },
  });

  pi.registerCommand("pi-memory-ingest", {
    description: "Manually ingest Pi sessions for the current project",
    handler: async (_args: string, ctx: any) => {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await ingestSessions({
          projectPath,
          storageBaseDir,
          trigger: "manual",
          sessionDir: runtimeConfig.sessionDirOverride,
        });

        ctx.ui?.notify?.(
          formatStatusBlock("Pi Memory ingest complete", [
            `run id: ${result.runId}`,
            `tracked sessions discovered: ${result.trackedSessionsDiscovered}`,
            `session files processed: ${result.sessionFilesProcessed}`,
            `entries seen: ${result.entriesSeen}`,
            `candidates found: ${result.candidatesFound}`,
            `memories created: ${result.memoriesCreated}`,
            `memories updated: ${result.memoriesUpdated}`,
            `memories ignored: ${result.memoriesIgnored}`,
            `last ingested at: ${result.lastIngestedAt || "never"}`,
          ]),
          "info",
        );
      } catch (error) {
        handleError(ctx, error, "Failed to ingest Pi sessions.");
      }
    },
  });

  pi.registerCommand("pi-memory-list", {
    description: "List stored Pi Memory items for the current project",
    handler: async (_args: string, ctx: any) => {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await listMemories({ projectPath, storageBaseDir, status: "active", limit: 50 });
        ctx.ui?.notify?.(formatMemoryRows("Pi Memory items", result.items), "info");
      } catch (error) {
        handleError(ctx, error, "Failed to list Pi memories.");
      }
    },
  });

  pi.registerCommand("pi-memory-search", {
    description: "Search stored Pi Memory items for the current project",
    handler: async (args: string, ctx: any) => {
      const query = args.trim();
      if (!query) {
        ctx.ui?.notify?.("Usage: /pi-memory-search <query>", "info");
        return;
      }

      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await searchMemories({ projectPath, storageBaseDir, query, limit: 20 });
        ctx.ui?.notify?.(formatMemoryRows(`Pi Memory search: ${query}`, result.items), "info");
      } catch (error) {
        handleError(ctx, error, "Failed to search Pi memories.");
      }
    },
  });

  pi.registerCommand("pi-memory-search-sessions", {
    description: "Search raw Pi session history for the current project",
    handler: async (args: string, ctx: any) => {
      const query = args.trim();
      if (!query) {
        ctx.ui?.notify?.("Usage: /pi-memory-search-sessions <query>", "info");
        return;
      }

      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      if (!runtimeConfig.rawSessionSearchEnabled) {
        ctx.ui?.notify?.("Raw session search is disabled by configuration.", "info");
        return;
      }

      try {
        const result = await searchSessions({
          projectPath,
          storageBaseDir,
          sessionDir: runtimeConfig.sessionDirOverride,
          query,
          limit: 20,
        });
        ctx.ui?.notify?.(formatSessionSearchRows(`Pi Memory raw session search: ${query}`, result.items), "info");
      } catch (error) {
        handleError(ctx, error, "Failed to search Pi sessions.");
      }
    },
  });

  pi.registerCommand("pi-memory-forget", {
    description: "Suppress a stored Pi Memory item by memory id",
    handler: async (args: string, ctx: any) => {
      const memoryId = args.trim();
      if (!memoryId) {
        ctx.ui?.notify?.("Usage: /pi-memory-forget <memoryId>", "info");
        return;
      }

      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await forgetMemory({
          projectPath,
          storageBaseDir,
          memoryId,
          mode: "suppressed",
        });
        ctx.ui?.notify?.(
          formatStatusBlock("Pi Memory item suppressed", [
            `memory id: ${result.memoryId}`,
            `status: ${result.status}`,
            `updated at: ${result.updatedAt}`,
          ]),
          "info",
        );
      } catch (error) {
        handleError(ctx, error, "Failed to forget Pi memory.");
      }
    },
  });

  pi.registerCommand("pi-memory-remember", {
    description: "Store an explicit Pi Memory item for the current project",
    handler: async (args: string, ctx: any) => {
      const text = args.trim();
      if (!text) {
        ctx.ui?.notify?.("Usage: /pi-memory-remember <text>", "info");
        return;
      }

      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await rememberMemory({
          projectPath,
          storageBaseDir,
          text,
        });
        ctx.ui?.notify?.(
          formatStatusBlock(result.created ? "Pi Memory item saved" : "Pi Memory item updated", [
            `memory id: ${result.memoryId}`,
            `category: ${result.category}`,
            `summary: ${result.summary}`,
            `status: ${result.status}`,
            `confidence: ${result.confidence.toFixed(2)}`,
            `importance: ${result.importance.toFixed(2)}`,
          ]),
          "info",
        );
      } catch (error) {
        handleError(ctx, error, "Failed to remember Pi memory.");
      }
    },
  });

  pi.registerCommand("pi-memory-rebuild", {
    description: "Rebuild derived project memory by clearing and re-ingesting sessions",
    handler: async (_args: string, ctx: any) => {
      const { projectPath, storageBaseDir } = resolveProjectContext(ctx.cwd);

      try {
        const result = await rebuildProjectMemory({
          projectPath,
          storageBaseDir,
          sessionDir: runtimeConfig.sessionDirOverride,
          trigger: "manual",
          activeSessionFile: ctx.sessionManager?.getSessionFile?.(),
        });
        ctx.ui?.notify?.(
          formatStatusBlock("Pi Memory rebuild complete", [
            `cleared memory sources: ${result.clearedMemorySources}`,
            `cleared memory items: ${result.clearedMemoryItems}`,
            `cleared ingestion state: ${result.clearedIngestionState}`,
            `cleared ingestion runs: ${result.clearedIngestionRuns}`,
            `run id: ${result.ingest.runId}`,
            `tracked sessions discovered: ${result.ingest.trackedSessionsDiscovered}`,
            `session files processed: ${result.ingest.sessionFilesProcessed}`,
            `entries seen: ${result.ingest.entriesSeen}`,
            `candidates found: ${result.ingest.candidatesFound}`,
            `memories created: ${result.ingest.memoriesCreated}`,
            `memories updated: ${result.ingest.memoriesUpdated}`,
            `memories ignored: ${result.ingest.memoriesIgnored}`,
          ]),
          "info",
        );
      } catch (error) {
        handleError(ctx, error, "Failed to rebuild Pi memory.");
      }
    },
  });
}

async function notifyAutoRecall(options: {
  ctx: any;
  projectPath: string;
  storageBaseDir: string;
  runtimeConfig: ReturnType<typeof resolveRuntimeBehaviorConfig>;
}): Promise<void> {
  if (!options.runtimeConfig.autoRecall) {
    return;
  }

  const recall = await recallMemories({
    projectPath: options.projectPath,
    storageBaseDir: options.storageBaseDir,
    limit: options.runtimeConfig.recallLimit,
  });

  if (recall.items.length > 0) {
    options.ctx.ui?.notify?.(formatMemoryRows("Relevant project memory", recall.items), "info");
  }
}

async function ensureProjectInitialized(projectPath: string, storageBaseDir: string): Promise<void> {
  const status = await getProjectStatus({ projectPath, storageBaseDir });
  if (!status.initialized) {
    throw new Error("Pi Memory is not initialized for this project. Run /pi-memory-init first.");
  }
}

function handleError(ctx: any, error: unknown, fallbackMessage: string) {
  if (error instanceof BackendError) {
    const stderr = typeof error.details?.stderr === "string" && error.details.stderr.trim()
      ? `\nstderr: ${error.details.stderr.trim()}`
      : "";
    ctx.ui?.notify?.(`${fallbackMessage} [${error.code}] ${error.message}${stderr}`, "error");
    return;
  }
  if (error instanceof Error) {
    ctx.ui?.notify?.(`${fallbackMessage} ${error.message}`, "error");
    return;
  }
  ctx.ui?.notify?.(fallbackMessage, "error");
}
