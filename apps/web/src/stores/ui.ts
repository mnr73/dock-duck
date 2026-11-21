import { useStorage } from "@vueuse/core";
import { defineStore } from "pinia";

/** Persisted UI preferences (localStorage via VueUse). */
export const useUiStore = defineStore("ui", () => {
    const sidebarCollapsed = useStorage("dockduck.sidebar.collapsed", false);
    const expandedProjects = useStorage<string[]>(
        "dockduck.sidebar.projects",
        [],
    );
    const expandedServices = useStorage<string[]>(
        "dockduck.sidebar.services",
        [],
    );

    function toggle(list: string[], key: string): string[] {
        return list.includes(key)
            ? list.filter((k) => k !== key)
            : [...list, key];
    }

    return {
        sidebarCollapsed,
        expandedProjects,
        expandedServices,
        toggleProject: (id: string) =>
            (expandedProjects.value = toggle(expandedProjects.value, id)),
        toggleService: (key: string) =>
            (expandedServices.value = toggle(expandedServices.value, key)),
    };
});
