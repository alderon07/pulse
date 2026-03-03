import { z } from "zod";

import {
  alertChannelSchema,
  checkSchema,
  createAlertChannelRequestSchema,
  createCheckRequestSchema,
  errorResponseSchema,
  listAlertChannelsResponseSchema,
  listChecksResponseSchema,
  listEventsResponseSchema,
  patchAlertChannelRequestSchema,
  patchCheckRequestSchema,
  pingBodySchema,
  pingResponseSchema,
  type CreateAlertChannelRequest,
  type CreateCheckRequest,
  type PatchAlertChannelRequest,
  type PatchCheckRequest,
  type PingBody,
} from "@/lib/api/schemas";

const DEFAULT_API_BASE_URL = "http://localhost:8080";

function getApiBaseUrl() {
  return (process.env.NEXT_PUBLIC_API_BASE_URL ?? DEFAULT_API_BASE_URL).replace(/\/$/, "");
}

type ApiErrorOptions = {
  status: number;
  code?: string;
  details?: unknown;
};

export class ApiError extends Error {
  status: number;
  code?: string;
  details?: unknown;

  constructor(message: string, options: ApiErrorOptions) {
    super(message);
    this.name = "ApiError";
    this.status = options.status;
    this.code = options.code;
    this.details = options.details;
  }
}

type ApiRequestOptions = {
  body?: unknown;
  headers?: HeadersInit;
  method?: "GET" | "POST" | "PATCH";
  token?: string;
};

async function parseResponseBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";

  if (contentType.includes("application/json")) {
    try {
      return await response.json();
    } catch {
      return null;
    }
  }

  const rawText = await response.text();
  return rawText ? { message: rawText } : null;
}

async function request<TSchema extends z.ZodTypeAny>(
  path: string,
  schema: TSchema,
  options: ApiRequestOptions = {}
): Promise<z.infer<TSchema>> {
  const headers = new Headers(options.headers);

  if (options.body !== undefined && !headers.has("content-type")) {
    headers.set("content-type", "application/json");
  }

  if (options.token) {
    headers.set("authorization", `Bearer ${options.token}`);
  }

  const response = await fetch(`${getApiBaseUrl()}${path}`, {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    cache: "no-store",
  });

  const payload = await parseResponseBody(response);

  if (!response.ok) {
    const parsedError = errorResponseSchema.safeParse(payload);
    if (parsedError.success) {
      throw new ApiError(parsedError.data.message, {
        status: response.status,
        code: parsedError.data.code,
        details: parsedError.data,
      });
    }

    throw new ApiError(`Request failed (${response.status})`, {
      status: response.status,
      details: payload,
    });
  }

  const parsed = schema.safeParse(payload);
  if (!parsed.success) {
    throw new ApiError("Response schema validation failed", {
      status: response.status,
      details: parsed.error.flatten(),
    });
  }

  return parsed.data;
}

type PaginationOptions = {
  limit?: number;
  offset?: number;
};

function withQuery(path: string, params: URLSearchParams) {
  const query = params.toString();
  return query ? `${path}?${query}` : path;
}

export async function pingCheck(token: string, body?: PingBody) {
  const parsedBody = body ? pingBodySchema.parse(body) : undefined;

  return request(`/v1/ping/${encodeURIComponent(token)}`, pingResponseSchema, {
    method: body ? "POST" : "GET",
    body: parsedBody,
  });
}

export async function listChecks(token: string, options: PaginationOptions = {}) {
  const params = new URLSearchParams();
  if (options.limit !== undefined) {
    params.set("limit", String(options.limit));
  }
  if (options.offset !== undefined) {
    params.set("offset", String(options.offset));
  }

  return request(withQuery("/api/v1/checks", params), listChecksResponseSchema, { token });
}

export async function createCheck(token: string, body: CreateCheckRequest) {
  const parsedBody = createCheckRequestSchema.parse(body);

  return request("/api/v1/checks", checkSchema, {
    method: "POST",
    token,
    body: parsedBody,
  });
}

export async function getCheck(token: string, id: string) {
  return request(`/api/v1/checks/${encodeURIComponent(id)}`, checkSchema, {
    token,
  });
}

export async function patchCheck(token: string, id: string, body: PatchCheckRequest) {
  const parsedBody = patchCheckRequestSchema.parse(body);

  return request(`/api/v1/checks/${encodeURIComponent(id)}`, checkSchema, {
    method: "PATCH",
    token,
    body: parsedBody,
  });
}

export async function pauseCheck(token: string, id: string) {
  return request(`/api/v1/checks/${encodeURIComponent(id)}/pause`, checkSchema, {
    method: "POST",
    token,
  });
}

export async function resumeCheck(token: string, id: string) {
  return request(`/api/v1/checks/${encodeURIComponent(id)}/resume`, checkSchema, {
    method: "POST",
    token,
  });
}

export async function rotateCheckToken(token: string, id: string) {
  return request(
    `/api/v1/checks/${encodeURIComponent(id)}/rotate-token`,
    checkSchema,
    {
      method: "POST",
      token,
    }
  );
}

export async function listCheckEvents(token: string, id: string, limit?: number) {
  const params = new URLSearchParams();
  if (limit !== undefined) {
    params.set("limit", String(limit));
  }

  return request(
    withQuery(`/api/v1/checks/${encodeURIComponent(id)}/events`, params),
    listEventsResponseSchema,
    {
      token,
    }
  );
}

export async function listAlertChannels(token: string, options: PaginationOptions = {}) {
  const params = new URLSearchParams();
  if (options.limit !== undefined) {
    params.set("limit", String(options.limit));
  }
  if (options.offset !== undefined) {
    params.set("offset", String(options.offset));
  }

  return request(withQuery("/api/v1/alert-channels", params), listAlertChannelsResponseSchema, {
    token,
  });
}

export async function createAlertChannel(token: string, body: CreateAlertChannelRequest) {
  const parsedBody = createAlertChannelRequestSchema.parse(body);

  return request("/api/v1/alert-channels", alertChannelSchema, {
    method: "POST",
    token,
    body: parsedBody,
  });
}

export async function patchAlertChannel(token: string, id: string, body: PatchAlertChannelRequest) {
  const parsedBody = patchAlertChannelRequestSchema.parse(body);

  return request(`/api/v1/alert-channels/${encodeURIComponent(id)}`, alertChannelSchema, {
    method: "PATCH",
    token,
    body: parsedBody,
  });
}
