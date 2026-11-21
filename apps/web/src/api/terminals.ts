import { api } from "./client";
import type {
    SystemTerminalStatus,
    TerminalInfo,
    TerminalOpenRequest,
} from "./types";

export const terminalsApi = {
    open: (project: string, req: TerminalOpenRequest) =>
        api.post<TerminalInfo>(`/projects/${project}/terminals`, req),
    list: () => api.get<TerminalInfo[]>("/terminals"),
    close: (id: string) => api.del(`/terminals/${id}`),
    system: () => api.get<SystemTerminalStatus>("/terminals/system"),
    openSystem: (project: string, req: TerminalOpenRequest) =>
        api.post<{ terminal: string; container: string; display: string }>(
            `/projects/${project}/terminals/system`,
            req,
        ),
};

/** WebSocket URL of a session (same host as the page, so the Vite proxy and a reverse proxy both work). */
export function terminalSocketUrl(id: string): string {
    const proto = location.protocol === "https:" ? "wss" : "ws";
    return `${proto}://${location.host}/ws/v1/terminals/${id}`;
}
