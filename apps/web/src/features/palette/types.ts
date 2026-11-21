import type { Component } from "vue";

export interface Command {
    id: string;
    title: string;
    /** muted text on the right (a path, a shortcut) */
    hint?: string;
    group: string;
    icon: Component;
    /** extra words the search also looks at */
    keywords?: string[];
    run: () => void;
}
