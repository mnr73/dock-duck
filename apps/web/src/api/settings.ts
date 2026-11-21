import { api } from "./client";
import type { ServerSettings, ServerSettingsPatch } from "./types";

export const settingsApi = {
    get: () => api.get<ServerSettings>("/settings"),
    /** only the given fields change */
    update: (patch: ServerSettingsPatch) =>
        api.put<ServerSettings>("/settings", patch),
};
