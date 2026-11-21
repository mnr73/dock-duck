import { createFetch } from "@vueuse/core";

/** Error with the server's `{error:{code,message}}` envelope (or a network failure). */
export class ApiError extends Error {
    constructor(
        public code: string,
        message: string,
        public status: number,
    ) {
        super(message);
        this.name = "ApiError";
    }
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
    const f = useApiFetch(path, {
        method,
        body: body === undefined ? undefined : JSON.stringify(body),
        headers:
            body === undefined
                ? headers
                : { "Content-Type": "application/json", ...headers },
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
    if (status >= 400) {
        const e = (
            parsed as { error?: { code?: string; message?: string } } | null
        )?.error;
        throw new ApiError(
            e?.code ?? "http_error",
            e?.message ?? `HTTP ${status}`,
            status,
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
