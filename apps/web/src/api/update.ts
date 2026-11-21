import { api } from "./client";
import type { UpdateStatus } from "./types";

export const updateApi = {
    status: (refresh = false) =>
        api.get<UpdateStatus>(`/update${refresh ? "?refresh=1" : ""}`),
    install: () =>
        api.post<{ installed: string; restart: "self" | "service" }>(
            "/update/install",
        ),
};
