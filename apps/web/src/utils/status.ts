import type { ContainerInfo, ProjectStatus, ServiceStatus } from "@/api/types";

/** Visual tone shared by every status dot / label. */
export type Tone = "ok" | "warn" | "err" | "off";

export function projectTone(s: ProjectStatus): Tone {
    switch (s) {
        case "running":
            return "ok";
        case "partial":
            return "warn";
        default:
            return "off";
    }
}

export function serviceTone(s: ServiceStatus): Tone {
    switch (s) {
        case "running":
            return "ok";
        case "starting":
        case "restarting":
        case "paused":
            return "warn";
        case "unhealthy":
        case "error":
            return "err";
        default:
            return "off";
    }
}

export function containerTone(
    c: Pick<ContainerInfo, "state" | "health" | "exit_code">,
): Tone {
    switch (c.state) {
        case "running":
            return c.health === "unhealthy"
                ? "err"
                : c.health === "starting"
                  ? "warn"
                  : "ok";
        case "restarting":
        case "paused":
            return "warn";
        case "exited":
        case "dead":
            return c.exit_code !== 0 ? "err" : "off";
        default:
            return "off";
    }
}

export const toneText: Record<Tone, string> = {
    ok: "text-ok-fg",
    warn: "text-warn-fg",
    err: "text-err-fg",
    off: "text-dim",
};

export function projectLabel(
    s: ProjectStatus,
    running: number,
    total: number,
): string {
    switch (s) {
        case "running":
            return `Running · ${running} of ${total} up`;
        case "partial":
            return `Degraded · ${running} of ${total} up`;
        case "stopped":
            return "Stopped";
        default:
            return "Not created yet";
    }
}

/** `/home/me/prj/x` → `~/prj/x` */
export function shortPath(p: string, home: string): string {
    if (home && (p === home || p.startsWith(home + "/"))) {
        return "~" + p.slice(home.length);
    }
    return p;
}

/** Path of `p` below `root` (or `p` itself when it is not inside). */
export function relativeTo(p: string, root: string | undefined): string {
    if (root && p.startsWith(root + "/")) {
        return p.slice(root.length + 1);
    }
    return p;
}

export function basename(p: string): string {
    return p.split("/").filter(Boolean).pop() ?? p;
}
