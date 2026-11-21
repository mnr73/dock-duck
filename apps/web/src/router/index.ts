import { createRouter, createWebHistory } from "vue-router";

export const router = createRouter({
    history: createWebHistory(),
    routes: [
        {
            path: "/",
            name: "home",
            component: () => import("@/pages/HomePage.vue"),
        },
        {
            path: "/projects/:id",
            component: () => import("@/pages/ProjectLayout.vue"),
            children: [
                {
                    path: "",
                    name: "project",
                    component: () => import("@/pages/ProjectServicesPage.vue"),
                },
                {
                    path: "services",
                    name: "project-services",
                    component: () => import("@/pages/ProjectServicesPage.vue"),
                },
                {
                    path: "compose",
                    name: "project-compose",
                    component: () => import("@/pages/ProjectComposePage.vue"),
                },
                {
                    path: "env",
                    name: "project-env",
                    component: () => import("@/pages/ProjectEnvPage.vue"),
                },
                {
                    path: "logs",
                    name: "project-logs",
                    component: () => import("@/pages/ProjectLogsPage.vue"),
                },
                {
                    path: "terminal",
                    name: "project-terminal",
                    component: () => import("@/pages/ProjectTerminalPage.vue"),
                },
            ],
        },
        {
            path: "/contexts",
            name: "contexts",
            component: () => import("@/pages/ContextsPage.vue"),
        },
        {
            path: "/roots",
            name: "roots",
            component: () => import("@/pages/RootsPage.vue"),
        },
        {
            path: "/ports",
            name: "ports",
            component: () => import("@/pages/PortsPage.vue"),
        },
        {
            path: "/registries",
            name: "registries",
            component: () => import("@/pages/RegistriesPage.vue"),
        },
        {
            path: "/settings",
            name: "settings",
            component: () => import("@/pages/SettingsPage.vue"),
        },
        { path: "/:pathMatch(.*)*", redirect: "/" },
    ],
});
