import { computed, ref } from "vue";

import { defineStore } from "pinia";

import { api } from "@/api/client";
import type { Health } from "@/api/types";
import { shortPath } from "@/utils/status";

export const useSystemStore = defineStore("system", () => {
    const health = ref<Health | null>(null);
    const serverDown = ref(false);

    const home = computed(() => health.value?.home ?? "");
    const version = computed(() => health.value?.version ?? "");

    async function load() {
        try {
            health.value = await api.get<Health>("/health");
            serverDown.value = false;
        } catch {
            serverDown.value = true;
        }
    }

    const short = (p: string) => shortPath(p, home.value);

    return { health, serverDown, home, version, load, short };
});
