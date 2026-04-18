import { access } from "node:fs/promises";
import { constants } from "node:fs";
import { spawn } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import type {
  GetProjectResult,
  ProjectInitResult,
  ProjectStatusResult,
} from "../types/project.ts";
import type { IngestSessionsResult } from "../types/ingest.ts";
import type { ListMemoriesResult, SearchMemoriesResult } from "../types/memory.ts";

const here = dirname(fileURLToPath(import.meta.url));
const projectRoot = resolve(here, "../../..");

export interface BackendRequest {
  version: 1;
  command: string;
  payload: Record<string, unknown>;
}

export interface BackendSuccess<Result = unknown> {
  ok: true;
  result: Result;
}

export interface BackendFailure {
  ok: false;
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

export type BackendResponse<Result = unknown> = BackendSuccess<Result> | BackendFailure;

export class BackendError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "BackendError";
  }
}

export async function callBackend<Result>(request: BackendRequest): Promise<Result> {
  const backendPath = await resolveBackendPath();
  const stdout = await runBackend(backendPath, JSON.stringify(request));

  const response = JSON.parse(stdout) as BackendResponse<Result>;
  if (!response.ok) {
    throw new BackendError(response.error.code, response.error.message, response.error.details);
  }
  return response.result;
}

export async function initProject(payload: {
  projectPath: string;
  storageBaseDir: string;
  projectName?: string;
}): Promise<ProjectInitResult> {
  return callBackend<ProjectInitResult>({ version: 1, command: "init_project", payload });
}

export async function getProject(payload: {
  projectPath: string;
  storageBaseDir: string;
}): Promise<GetProjectResult> {
  return callBackend<GetProjectResult>({ version: 1, command: "get_project", payload });
}

export async function getProjectStatus(payload: {
  projectPath: string;
  storageBaseDir: string;
}): Promise<ProjectStatusResult> {
  return callBackend<ProjectStatusResult>({ version: 1, command: "project_status", payload });
}

export async function ingestSessions(payload: {
  projectPath: string;
  storageBaseDir: string;
  sessionDir?: string;
  trigger?: string;
  activeSessionFile?: string;
}): Promise<IngestSessionsResult> {
  return callBackend<IngestSessionsResult>({ version: 1, command: "ingest_sessions", payload });
}

export async function listMemories(payload: {
  projectPath: string;
  storageBaseDir: string;
  status?: string;
  limit?: number;
}): Promise<ListMemoriesResult> {
  return callBackend<ListMemoriesResult>({ version: 1, command: "list_memories", payload });
}

export async function searchMemories(payload: {
  projectPath: string;
  storageBaseDir: string;
  query: string;
  limit?: number;
}): Promise<SearchMemoriesResult> {
  return callBackend<SearchMemoriesResult>({ version: 1, command: "search_memories", payload });
}

async function resolveBackendPath(): Promise<string> {
  const override = process.env.PI_MEMORY_BACKEND_PATH?.trim();
  if (override) {
    await assertExecutable(override);
    return override;
  }

  const candidatePaths = [
    join(projectRoot, "dist/package/bin/pi-memory-backend"),
    join(projectRoot, "resources/bin", `${mapPlatform()}-${mapArch()}`, binaryName()),
  ];

  for (const candidate of candidatePaths) {
    try {
      await assertExecutable(candidate);
      return candidate;
    } catch {
      // Try next candidate.
    }
  }

  throw new BackendError(
    "BACKEND_NOT_FOUND",
    "Could not locate the pi-memory backend binary. Build it first or set PI_MEMORY_BACKEND_PATH.",
    { candidatePaths },
  );
}

async function assertExecutable(path: string): Promise<void> {
  await access(path, constants.X_OK);
}

function mapPlatform(): string {
  switch (process.platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new BackendError("UNSUPPORTED_PLATFORM", `Unsupported platform: ${process.platform}`);
  }
}

function mapArch(): string {
  switch (process.arch) {
    case "arm64":
      return "arm64";
    case "x64":
      return "x64";
    default:
      throw new BackendError("UNSUPPORTED_ARCH", `Unsupported architecture: ${process.arch}`);
  }
}

function binaryName(): string {
  return process.platform === "win32" ? "pi-memory-backend.exe" : "pi-memory-backend";
}

function runBackend(binaryPath: string, input: string): Promise<string> {
  return new Promise((resolvePromise, reject) => {
    const child = spawn(binaryPath, [], {
      cwd: projectRoot,
      stdio: ["pipe", "pipe", "pipe"],
    });

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });

    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });

    child.on("error", (error) => {
      reject(error);
    });

    child.on("close", (code) => {
      if (code === 0) {
        resolvePromise(stdout);
        return;
      }
      reject(new BackendError("BACKEND_PROCESS_FAILED", stderr.trim() || `Backend exited with code ${code}`));
    });

    child.stdin.write(input);
    child.stdin.end();
  });
}
