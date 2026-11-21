import { api } from "./client";
import type { EnvView, FileContent, FileRef, Validation } from "./types";

const base = (project: string) => `/projects/${project}`;

export const filesApi = {
    list: (project: string) => api.get<FileRef[]>(`${base(project)}/files`),

    /** `reveal` returns real secret values of a .env file (otherwise they are masked). */
    read: (project: string, file: string, reveal = false) =>
        api.get<FileContent>(
            `${base(project)}/files/${file}${reveal ? "?reveal=1" : ""}`,
        ),

    /** `etag` is the one received with the content ("" = create a new .env). */
    save: (project: string, file: string, content: string, etag: string) =>
        api.put<{ etag: string; saved_at: string; backup: string }>(
            `${base(project)}/files/${file}`,
            { content },
            { "If-Match": etag },
        ),

    validate: (project: string, file: string, content: string) =>
        api.post<Validation>(`${base(project)}/files/${file}/validate`, {
            content,
        }),

    env: (project: string, reveal = false) =>
        api.get<EnvView>(`${base(project)}/env${reveal ? "?reveal=1" : ""}`),
};
