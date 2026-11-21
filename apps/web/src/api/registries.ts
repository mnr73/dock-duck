import { api } from "./client";
import type { RegistryLoginRequest, RegistryState } from "./types";

export const registriesApi = {
    state: () => api.get<RegistryState>("/registries"),
    login: (req: RegistryLoginRequest) =>
        api.post<{ server: string; plaintext: boolean }>(
            "/registries/login",
            req,
        ),
    logout: (server: string) =>
        api.post<void>("/registries/logout", { server }),
};
