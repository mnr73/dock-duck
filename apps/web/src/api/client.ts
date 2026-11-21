import { createFetch } from "@vueuse/core";

/** Typed-ish JSON client for /api/v1 (VueUse createFetch). */
export const useApi = createFetch({
    baseUrl: "/api/v1",
    options: { immediate: true },
    fetchOptions: { headers: { Accept: "application/json" } },
});

export interface Health {
    ok: boolean;
    version: string;
}
