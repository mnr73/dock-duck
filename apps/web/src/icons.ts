/**
 * All icons of the app (Iconify / Lucide) live in this one module.
 *
 *   import { IconPlay } from "@/icons";
 *
 * Need a new icon? Add it here under a *semantic* name (the concept, not the
 * glyph). Components never import from `~icons/*` directly (enforced by ESLint).
 * Each icon is a tiny module; unused re-exports are tree-shaken from the bundle.
 * Browse: https://icon-sets.iconify.design/lucide/
 */

/* ------------------------------- Status ----------------------------------- */
export { default as IconCheck } from "~icons/lucide/check";
export { default as IconClose } from "~icons/lucide/x";
export { default as IconError } from "~icons/lucide/circle-alert";
export { default as IconWarning } from "~icons/lucide/triangle-alert";
export { default as IconInfo } from "~icons/lucide/info";

/* ------------------------------- Actions ---------------------------------- */
export { default as IconPlay } from "~icons/lucide/play";
export { default as IconStop } from "~icons/lucide/square";
export { default as IconRestart } from "~icons/lucide/rotate-cw";
export { default as IconRefresh } from "~icons/lucide/refresh-cw";

/* ------------------------------- Brand / UI ------------------------------- */
export { default as IconDocker } from "~icons/lucide/container";
export { default as IconSidebar } from "~icons/lucide/panel-left";
export { default as IconSettings } from "~icons/lucide/settings";
