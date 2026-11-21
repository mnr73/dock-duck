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
                    component: () => import("@/pages/ComingSoonPage.vue"),
                    props: { title: "Compose editor", phase: "Phase 6" },
                },
                {
                    path: "env",
                    component: () => import("@/pages/ComingSoonPage.vue"),
                    props: { title: "Environment", phase: "Phase 6" },
                },
                {
                    path: "logs",
                    component: () => import("@/pages/ComingSoonPage.vue"),
                    props: { title: "Logs", phase: "Phase 5" },
                },
                {
                    path: "terminal",
                    component: () => import("@/pages/ComingSoonPage.vue"),
                    props: { title: "Terminal", phase: "Phase 7" },
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
            component: () => import("@/pages/ComingSoonPage.vue"),
            props: { title: "Registries", phase: "Phase 8" },
        },
        {
            path: "/settings",
            name: "settings",
            component: () => import("@/pages/SettingsPage.vue"),
        },
        { path: "/:pathMatch(.*)*", redirect: "/" },
    ],
});
