import { createFetch } from "@vueuse/core";

/** Error with the server's `{error:{code,message}}` envelope (or a network failure). */
export class ApiError extends Error {
    constructor(
        public code: string,
        message: string,
        public status: number,
        /** seconds from a Retry-After header (login lock, busy server) */
        public retryAfter = 0,
    ) {
        super(message);
        this.name = "ApiError";
    }
}

/** The CSRF token of the current login session; sent with every state-changing request. */
let csrfToken = "";
export function setCsrfToken(t: string) {
    csrfToken = t;
}

/** Called when the server says the session is gone (401): the app shows the login screen. */
let onUnauthorized: (() => void) | undefined;
export function setUnauthorizedHandler(fn: () => void) {
    onUnauthorized = fn;
}

const useApiFetch = createFetch({
    baseUrl: "/api/v1",
    options: { immediate: false, updateDataOnError: true },
    fetchOptions: { headers: { Accept: "application/json" } },
});

async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    headers: Record<string, string> = {},
): Promise<T> {
    const all: Record<string, string> = { ...headers };
    if (body !== undefined) all["Content-Type"] = "application/json";
    if (method !== "GET" && csrfToken) all["X-CSRF-Token"] = csrfToken;
    const f = useApiFetch(path, {
        method,
        body: body === undefined ? undefined : JSON.stringify(body),
        headers: all,
    });
    await f.execute();

    const status = f.statusCode.value ?? 0;
    const text = typeof f.data.value === "string" ? f.data.value : "";
    let parsed: unknown = null;
    if (text) {
        try {
            parsed = JSON.parse(text);
        } catch {
            parsed = null;
        }
    }

    if (status === 0) {
        throw new ApiError("network", "DockDuck server is unreachable", 0);
    }
    if (status === 401 && !path.startsWith("/auth/")) onUnauthorized?.();
    if (status >= 400) {
        const e = (
            parsed as { error?: { code?: string; message?: string } } | null
        )?.error;
        const ra = Number(f.response.value?.headers.get("Retry-After") ?? 0);
        throw new ApiError(
            e?.code ?? "http_error",
            e?.message ?? `HTTP ${status}`,
            status,
            Number.isFinite(ra) ? ra : 0,
        );
    }
    return parsed as T;
}

export const api = {
    get: <T>(path: string) => request<T>("GET", path),
    post: <T>(path: string, body?: unknown) =>
        request<T>("POST", path, body ?? {}),
    put: <T>(path: string, body?: unknown, headers?: Record<string, string>) =>
        request<T>("PUT", path, body ?? {}, headers),
    del: <T = void>(path: string) => request<T>("DELETE", path),
};

export function errorMessage(e: unknown): string {
    return e instanceof Error ? e.message : String(e);
}
