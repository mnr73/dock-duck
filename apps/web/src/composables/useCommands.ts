import { type ComputedRef, computed } from "vue";

import { useRoute, useRouter } from "vue-router";

import {
    IconBuild,
    IconCompose,
    IconContext,
    IconEnv,
    IconFolder,
    IconLogs,
    IconPlay,
    IconPorts,
    IconProjects,
    IconPull,
    IconRefresh,
    IconRegistry,
    IconRestart,
    IconSettings,
    IconStop,
    IconTerminal,
    IconThemeDark,
    IconThemeLight,
    IconThemeSystem,
} from "@/icons";

import type { Command } from "@/features/palette/types";

import { useJobsStore } from "@/stores/jobs";
import { useProjectsStore } from "@/stores/projects";
import { useSystemStore } from "@/stores/system";

import { useTheme } from "./useTheme";

/**
 * Everything the palette can do, built from the current state: pages, every
 * project, the open project's tabs and safe actions, and the theme.
 *
 * `down` is deliberately absent: removing containers goes through its own dialog
 * (and, for volumes, a typed confirmation), never a keystroke away.
 */
export function useCommands(): ComputedRef<Command[]> {
    const router = useRouter();
    const route = useRoute();
    const projects = useProjectsStore();
    const jobs = useJobsStore();
    const system = useSystemStore();
    const theme = useTheme();

    return computed<Command[]>(() => {
        const go = (to: string) => () => void router.push(to);
        const out: Command[] = [
            {
                id: "go-home",
                title: "Projects",
                group: "Go to",
                icon: IconProjects,
                run: go("/"),
            },
            {
                id: "go-contexts",
                title: "Docker contexts",
                group: "Go to",
                icon: IconContext,
                keywords: ["engine", "server"],
                run: go("/contexts"),
            },
            {
                id: "go-roots",
                title: "Root directories",
                group: "Go to",
                icon: IconFolder,
                keywords: ["folders", "scan", "add"],
                run: go("/roots"),
            },
            {
                id: "go-ports",
                title: "Ports in use",
                group: "Go to",
                icon: IconPorts,
                keywords: ["listening", "published"],
                run: go("/ports"),
            },
            {
                id: "go-registries",
                title: "Registries",
                group: "Go to",
                icon: IconRegistry,
                keywords: ["docker login", "hub", "sign in"],
                run: go("/registries"),
            },
            {
                id: "go-settings",
                title: "Settings",
                group: "Go to",
                icon: IconSettings,
                keywords: ["preferences", "password", "theme"],
                run: go("/settings"),
            },
        ];

        const id = typeof route.params.id === "string" ? route.params.id : "";
        const current = projects.projects.find((p) => p.id === id);
        if (current) {
            const base = `/projects/${current.id}`;
            const tabs: [string, string, typeof IconProjects, string][] = [
                ["services", "Services", IconProjects, base],
                ["compose", "Compose file", IconCompose, `${base}/compose`],
                ["env", "Environment (.env)", IconEnv, `${base}/env`],
                ["logs", "Logs", IconLogs, `${base}/logs`],
                ["terminal", "Terminal", IconTerminal, `${base}/terminal`],
            ];
            for (const [key, title, icon, to] of tabs)
                out.push({
                    id: `tab-${key}`,
                    title: `${current.name}: ${title}`,
                    group: "This project",
                    icon,
                    keywords: [key],
                    run: go(to),
                });

            const act = (action: string) => () =>
                void jobs.run({
                    projectId: current.id,
                    projectName: current.name,
                    action,
                });
            const actions: [string, string, typeof IconPlay][] = [
                ["up", "Up", IconPlay],
                ["restart", "Restart", IconRestart],
                ["stop", "Stop", IconStop],
                ["pull", "Pull images", IconPull],
            ];
            for (const [a, label, icon] of actions)
                out.push({
                    id: `act-${a}`,
                    title: `${label} ${current.name}`,
                    group: "This project",
                    icon,
                    keywords: [a],
                    run: act(a),
                });
            out.push({
                id: "act-build",
                title: `Build ${current.name}…`,
                group: "This project",
                icon: IconBuild,
                keywords: ["build"],
                run: go(base),
            });
        }

        for (const p of projects.projects) {
            out.push({
                id: `project-${p.id}`,
                title: p.name,
                hint: system.short(p.dir),
                group: "Projects",
                icon: IconProjects,
                keywords: [p.dir],
                run: go(`/projects/${p.id}`),
            });
        }

        out.push({
            id: "rescan",
            title: "Rescan roots",
            group: "Actions",
            icon: IconRefresh,
            keywords: ["refresh", "discover"],
            run: () => void projects.rescan(),
        });
        for (const [value, label, icon] of [
            ["auto", "System", IconThemeSystem],
            ["light", "Light", IconThemeLight],
            ["dark", "Dark", IconThemeDark],
        ] as const)
            out.push({
                id: `theme-${value}`,
                title: `Theme: ${label}`,
                group: "Appearance",
                icon,
                keywords: ["color", "dark mode", "light mode"],
                run: () => (theme.value = value),
            });
        return out;
    });
}
