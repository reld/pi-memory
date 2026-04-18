import { BackendError, getProjectStatus, ingestSessions, initProject } from "./services/backend.ts";
import { resolveProjectContext } from "./services/project.ts";
import { formatStatusBlock } from "./util/formatting.ts";

export default function createPiMemoryExtension(pi: any) {
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
            `active memories: ${status.activeMemoryCount}`,
            `tracked sessions: ${status.trackedSessionCount}`,
            `last ingested at: ${status.lastIngestedAt || "never"}`,
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
          sessionDir: process.env.PI_MEMORY_SESSION_DIR,
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
}

function handleError(ctx: any, error: unknown, fallbackMessage: string) {
  if (error instanceof BackendError) {
    ctx.ui?.notify?.(`${fallbackMessage} [${error.code}] ${error.message}`, "error");
    return;
  }
  if (error instanceof Error) {
    ctx.ui?.notify?.(`${fallbackMessage} ${error.message}`, "error");
    return;
  }
  ctx.ui?.notify?.(fallbackMessage, "error");
}
