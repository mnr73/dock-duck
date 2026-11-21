import { createSharedComposable, useColorMode } from "@vueuse/core";

export type ThemeChoice = "auto" | "light" | "dark";

/** System / Light / Dark. VueUse toggles the `dark` class on <html> and persists the choice. */
export const useTheme = createSharedComposable(() => {
    const choice = useColorMode<ThemeChoice>({
        selector: "html",
        attribute: "class",
        modes: { light: "light", dark: "dark" },
        storageKey: "dockduck.theme",
        initialValue: "auto",
    });
    return choice;
});
