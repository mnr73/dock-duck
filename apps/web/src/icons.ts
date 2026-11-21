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
export { default as IconSuccess } from "~icons/lucide/circle-check";
export { default as IconSpinner } from "~icons/lucide/loader-circle";
export { default as IconDisconnected } from "~icons/lucide/unplug";

/* ------------------------------- Actions ---------------------------------- */
export { default as IconPlay } from "~icons/lucide/play";
export { default as IconStop } from "~icons/lucide/square";
export { default as IconRestart } from "~icons/lucide/rotate-cw";
/** compose down: stop and remove */
export { default as IconDown } from "~icons/lucide/circle-arrow-down";
export { default as IconRefresh } from "~icons/lucide/refresh-cw";
export { default as IconAdd } from "~icons/lucide/plus";
export { default as IconRemove } from "~icons/lucide/trash-2";
export { default as IconMore } from "~icons/lucide/ellipsis";
export { default as IconSearch } from "~icons/lucide/search";
export { default as IconExternal } from "~icons/lucide/external-link";

export { default as IconCopy } from "~icons/lucide/copy";
export { default as IconInspect } from "~icons/lucide/scan-search";
export { default as IconPause } from "~icons/lucide/pause";

/* ------------------------------- Navigation ------------------------------- */
export { default as IconChevronRight } from "~icons/lucide/chevron-right";
export { default as IconChevronDown } from "~icons/lucide/chevron-down";
export { default as IconSidebarOpen } from "~icons/lucide/panel-left-open";
export { default as IconSidebarClose } from "~icons/lucide/panel-left-close";

/* ------------------------------- Domain ----------------------------------- */
/** Compose project (stack of services) */
export { default as IconProjects } from "~icons/lucide/layers";
/** Docker context / engine */
export { default as IconContext } from "~icons/lucide/server";
/** Project root directory */
export { default as IconFolder } from "~icons/lucide/folder";
export { default as IconFolderAdd } from "~icons/lucide/folder-plus";
/** Registry credentials */
export { default as IconRegistry } from "~icons/lucide/key-round";
export { default as IconSettings } from "~icons/lucide/settings";
/** Compose service */
export { default as IconService } from "~icons/lucide/box";
/** A running container instance */
export { default as IconContainer } from "~icons/lucide/container";
export { default as IconLogo } from "~icons/lucide/bird";

/* ------------------------------- Theme ------------------------------------ */
export { default as IconThemeLight } from "~icons/lucide/sun";
export { default as IconThemeDark } from "~icons/lucide/moon";
export { default as IconThemeSystem } from "~icons/lucide/monitor";

/* ------------------------------- Tabs ------------------------------------- */
export { default as IconCompose } from "~icons/lucide/file-code";
export { default as IconEnv } from "~icons/lucide/file-cog";
export { default as IconLogs } from "~icons/lucide/scroll-text";
export { default as IconTerminal } from "~icons/lucide/square-terminal";
export { default as IconPorts } from "~icons/lucide/cable";
