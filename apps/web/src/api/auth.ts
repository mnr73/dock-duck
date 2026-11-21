import { api } from "./client";
import type { AuthStatus } from "./types";

export const authApi = {
    status: () => api.get<AuthStatus>("/auth/status"),
    setup: (password: string) =>
        api.post<{ csrf: string }>("/auth/setup", { password }),
    login: (password: string) =>
        api.post<{ csrf: string }>("/auth/login", { password }),
    logout: () => api.post<{ authenticated: boolean }>("/auth/logout"),
    changePassword: (current: string, next: string) =>
        api.post<{ csrf: string }>("/auth/password", { current, next }),
};
