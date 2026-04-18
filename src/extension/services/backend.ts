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
