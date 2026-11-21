import { computed, ref } from "vue";

import { useStorage } from "@vueuse/core";
import { defineStore } from "pinia";

import { errorMessage } from "@/api/client";
import { terminalsApi } from "@/api/terminals";
import type { TerminalInfo, TerminalOpenRequest } from "@/api/types";

export type UserChoice = "default" | "root" | "uid1000" | "custom";

export interface TerminalPrefs {
    user: UserChoice;
    customUser: string;
    shell: string; // "" = auto (bash, then sh)
}

/**
 * Terminal sessions live on the server; this store mirrors them per project so the
 * tab bar survives page changes and reloads (the shells keep running meanwhile).
 */
export const useTerminalsStore = defineStore("terminals", () => {
    const sessions = ref<TerminalInfo[]>([]);
    const loading = ref(false);
    const error = ref("");

    // defaults for the "new terminal" dialog (Phase 9 moves these into Settings)
    const prefs = useStorage<TerminalPrefs>("dockduck.terminal.prefs", {
        user: "default",
        customUser: "",
        shell: "",
    });
    const fontSize = useStorage("dockduck.terminal.fontSize", 13);

    const byProject = (projectId: string) =>
        computed(() => sessions.value.filter((s) => s.project === projectId));

    async function refresh() {
        loading.value = true;
        try {
            sessions.value = await terminalsApi.list();
            error.value = "";
        } catch (e) {
            error.value = errorMessage(e);
        } finally {
            loading.value = false;
        }
    }

    function userArg(p: TerminalPrefs): string | undefined {
        switch (p.user) {
            case "root":
                return "0";
            case "uid1000":
                return "1000";
            case "custom":
                return p.customUser.trim() || undefined;
            default:
                return undefined;
        }
    }

    async function open(
        projectId: string,
        req: TerminalOpenRequest,
    ): Promise<TerminalInfo> {
        const info = await terminalsApi.open(projectId, req);
        sessions.value = [...sessions.value, info];
        return info;
    }

    async function close(id: string) {
        await terminalsApi.close(id).catch(() => undefined);
        sessions.value = sessions.value.filter((s) => s.id !== id);
    }

    function markExited(id: string, code: number | null) {
        sessions.value = sessions.value.map((s) =>
            s.id === id
                ? { ...s, state: "exited", exit_code: code ?? undefined }
                : s,
        );
    }

    return {
        sessions,
        loading,
        error,
        prefs,
        fontSize,
        byProject,
        refresh,
        open,
        close,
        markExited,
        userArg,
    };
});
