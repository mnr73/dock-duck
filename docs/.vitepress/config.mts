import { defineConfig } from "vitepress";

// Served at https://mnr73.github.io/dock-duck/
export default defineConfig({
    title: "DockDuck",
    description:
        "A web GUI for the Docker Compose projects you already have. One binary, your own folders.",
    base: "/dock-duck/",
    cleanUrls: true,
    lastUpdated: true,
    head: [["link", { rel: "icon", href: "/dock-duck/favicon.svg" }]],
    themeConfig: {
        logo: "/logo.svg",
        nav: [
            { text: "Guide", link: "/guide/install" },
            { text: "Server setup", link: "/server/" },
            { text: "Reference", link: "/reference/api" },
            {
                text: "Releases",
                link: "https://github.com/mnr73/dock-duck/releases",
            },
        ],
        sidebar: [
            {
                text: "Getting started",
                items: [
                    { text: "What is DockDuck", link: "/" },
                    { text: "Install and update", link: "/guide/install" },
                    { text: "First run", link: "/guide/first-run" },
                ],
            },
            {
                text: "Using it",
                items: [
                    { text: "Projects and actions", link: "/guide/projects" },
                    { text: "Logs and terminal", link: "/guide/logs-terminal" },
                    { text: "Compose and .env", link: "/guide/editors" },
                    { text: "Registries", link: "/guide/registries" },
                    { text: "Settings and shortcuts", link: "/guide/settings" },
                    { text: "AI agent skill", link: "/guide/ai-skill" },
                ],
            },
            {
                text: "Running it on a server",
                items: [
                    { text: "Overview", link: "/server/" },
                    { text: "systemd service", link: "/server/systemd" },
                    { text: "Reverse proxy and TLS", link: "/server/reverse-proxy" },
                    { text: "Security", link: "/server/security" },
                ],
            },
            {
                text: "Reference",
                items: [
                    { text: "Command line", link: "/reference/cli" },
                    { text: "Configuration", link: "/reference/config" },
                    { text: "HTTP API", link: "/reference/api" },
                    { text: "Troubleshooting", link: "/reference/troubleshooting" },
                ],
            },
        ],
        socialLinks: [{ icon: "github", link: "https://github.com/mnr73/dock-duck" }],
        editLink: {
            pattern: "https://github.com/mnr73/dock-duck/edit/main/docs/:path",
        },
        search: { provider: "local" },
        footer: {
            message: "Released under the MIT License.",
            copyright: "Copyright © 2026 Milad Nazari",
        },
    },
});
