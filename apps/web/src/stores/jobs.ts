import { computed, ref } from "vue";

import { defineStore } from "pinia";
import { toast } from "vue-sonner";

import { api, errorMessage } from "@/api/client";
import type { JobInfo } from "@/api/types";

export interface ActionRequest {
    projectId: string;
    projectName: string;
    action: string;
    services?: string[];
    volumes?: boolean;
    removeOrphans?: boolean;
    build?: boolean;
    /** build: ignore the layer cache */
    noCache?: boolean;
    /** build: re-pull base images first */
    pull?: boolean;
}

const PAST: Record<string, string> = {
    up: "is up",
    down: "was taken down",
    start: "started",
    stop: "stopped",
    restart: "restarted",
    build: "built",
    pull: "images pulled",
};

/** Runs project actions as server jobs, follows their output over WebSocket and toasts the result. */
export const useJobsStore = defineStore("jobs", () => {
    const current = ref<JobInfo | null>(null);
    const projectName = ref("");
    const lines = ref<string[]>([]);
    const drawerOpen = ref(false);
    /** project id → running action, so buttons can show a spinner after reloads of the view */
    const running = ref<Record<string, string>>({});
    let socket: WebSocket | null = null;
    let onFinished: (() => void) | null = null;

    const isRunning = computed(() => current.value?.status === "running");

    function setOnFinished(fn: (() => void) | null) {
        onFinished = fn;
    }

    async function run(req: ActionRequest): Promise<boolean> {
        const label = req.services?.length
            ? `${req.action} ${req.services.join(", ")}`
            : req.action;
        try {
            const res = await api.post<{ job_id: string; job: JobInfo }>(
                `/projects/${req.projectId}/actions/${req.action}`,
                {
                    services: req.services,
                    volumes: req.volumes,
                    remove_orphans: req.removeOrphans,
                    build: req.build,
                    no_cache: req.noCache,
                    pull: req.pull,
                },
            );
            current.value = res.job;
            projectName.value = req.projectName;
            lines.value = [];
            running.value[req.projectId] = label;
            follow(res.job.id, req, label);
            return true;
        } catch (e) {
            toast.error(`${label} failed`, { description: errorMessage(e) });
            return false;
        }
    }

    function follow(jobId: string, req: ActionRequest, label: string) {
        socket?.close();
        const proto = location.protocol === "https:" ? "wss" : "ws";
        const ws = new WebSocket(
            `${proto}://${location.host}/ws/v1/jobs/${jobId}`,
        );
        socket = ws;
        let finished = false;

        const finish = async (
            status: "succeeded" | "failed",
            error: string,
        ) => {
            if (finished) return;
            finished = true;
            delete running.value[req.projectId];
            if (current.value?.id === jobId)
                current.value = { ...current.value, status, error };
            if (status === "succeeded") {
                toast.success(
                    `${req.projectName} ${req.services?.length ? label.replace(`${req.action} `, "") + " " : ""}${PAST[req.action] ?? "done"}`,
                );
            } else {
                toast.error(`${req.projectName}: ${label} failed`, {
                    description: lastUsefulLine(lines.value) || error,
                    action: {
                        label: "Show output",
                        onClick: () => (drawerOpen.value = true),
                    },
                    duration: 12000,
                });
            }
            onFinished?.();
        };

        ws.onmessage = (e) => {
            const text = String(e.data);
            const done = parseDone(text);
            if (done) {
                finish(
                    done.status === "succeeded" ? "succeeded" : "failed",
                    done.error,
                );
                return;
            }
            lines.value.push(text);
            if (lines.value.length > 3000)
                lines.value.splice(0, lines.value.length - 3000);
        };
        ws.onclose = async () => {
            if (finished) return;
            // connection dropped before "done": ask the server for the final state
            try {
                const j = await api.get<JobInfo>(`/jobs/${jobId}`);
                if (j.status === "running") {
                    setTimeout(() => follow(jobId, req, label), 1000);
                    return;
                }
                finish(
                    j.status === "succeeded" ? "succeeded" : "failed",
                    j.error ?? "",
                );
            } catch {
                finish("failed", "lost connection to the server");
            }
        };
    }

    return {
        current,
        projectName,
        lines,
        drawerOpen,
        running,
        isRunning,
        run,
        setOnFinished,
    };
});

/** Last non-empty line that looks like an error, else the last line. */
export function lastUsefulLine(lines: string[]): string {
    const clean = lines.map((l) => l.trim()).filter(Boolean);
    const err = [...clean]
        .reverse()
        .find((l) =>
            /error|failed|denied|conflict|already|cannot|can't|no such/i.test(
                l,
            ),
        );
    return err ?? clean[clean.length - 1] ?? "";
}

/** The server ends a job stream with a JSON control frame; everything else is an output line. */
export function parseDone(
    text: string,
): { status: string; error: string } | null {
    if (!text.startsWith("{") || !text.includes('"done"')) return null;
    try {
        const d = JSON.parse(text);
        if (d && d.type === "done")
            return {
                status: String(d.status ?? ""),
                error: String(d.error ?? ""),
            };
    } catch {
        /* a log line that merely looks like JSON */
    }
    return null;
}
