import { computed, ref } from "vue";

import { defineStore } from "pinia";

import { api, errorMessage } from "@/api/client";
import type { ProjectDetail, ProjectSummary, Root } from "@/api/types";
import { relativeTo } from "@/utils/status";

export interface ProjectGroup {
    key: string;
    /** absolute root path, or "" for projects known only to Docker */
    root: string;
    projects: ProjectSummary[];
}

export const useProjectsStore = defineStore("projects", () => {
    const projects = ref<ProjectSummary[]>([]);
    const roots = ref<Root[]>([]);
    const loading = ref(false);
    const loaded = ref(false);
    const error = ref<string | null>(null);
    const lastScan = ref<Date | null>(null);
    const details = ref<Record<string, ProjectDetail>>({});
    const detailLoading = ref<Record<string, boolean>>({});

    const groups = computed<ProjectGroup[]>(() => {
        const out: ProjectGroup[] = roots.value.map((r) => ({
            key: r.id,
            root: r.path,
            projects: [],
        }));
        const outside: ProjectGroup = {
            key: "outside",
            root: "",
            projects: [],
        };
        for (const p of projects.value) {
            (out.find((g) => g.root === p.root) ?? outside).projects.push(p);
        }
        if (outside.projects.length) out.push(outside);
        return out;
    });

    /** Names used by more than one project: the UI adds the path to tell them apart. */
    const duplicateNames = computed(() => {
        const seen = new Map<string, number>();
        for (const p of projects.value)
            seen.set(p.name, (seen.get(p.name) ?? 0) + 1);
        return new Set(
            [...seen].filter(([, n]) => n > 1).map(([name]) => name),
        );
    });

    const runningContainers = computed(() =>
        projects.value.reduce((n, p) => n + p.running, 0),
    );
    const totalContainers = computed(() =>
        projects.value.reduce((n, p) => n + p.total, 0),
    );

    const byId = (id: string) => projects.value.find((p) => p.id === id);
    const hint = (p: ProjectSummary) =>
        duplicateNames.value.has(p.name) ? relativeTo(p.dir, p.root) : "";

    /** quiet = keep the current list on screen while refreshing (live events). */
    async function load(quiet = false) {
        if (!quiet) loading.value = true;
        try {
            const [r, p] = await Promise.all([
                api.get<Root[] | null>("/roots"),
                api.get<ProjectSummary[]>("/projects"),
            ]);
            roots.value = r ?? [];
            projects.value = p ?? [];
            error.value = null;
        } catch (e) {
            error.value = errorMessage(e);
        } finally {
            loading.value = false;
            loaded.value = true;
        }
    }

    async function rescan() {
        await api.post("/roots/rescan");
        lastScan.value = new Date();
        await load(true);
    }

    async function addRoot(path: string) {
        await api.post("/roots", { path });
        lastScan.value = new Date();
        await load(true);
    }

    async function removeRoot(id: string) {
        await api.del(`/roots/${id}`);
        await load(true);
    }

    async function loadDetail(id: string, quiet = false) {
        if (!quiet) detailLoading.value[id] = true;
        try {
            details.value[id] = await api.get<ProjectDetail>(`/projects/${id}`);
        } finally {
            detailLoading.value[id] = false;
        }
    }

    function resetDetails() {
        details.value = {};
    }

    return {
        projects,
        roots,
        loading,
        loaded,
        error,
        lastScan,
        details,
        detailLoading,
        groups,
        duplicateNames,
        runningContainers,
        totalContainers,
        byId,
        hint,
        load,
        rescan,
        addRoot,
        removeRoot,
        loadDetail,
        resetDetails,
    };
});
