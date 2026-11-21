<script setup lang="ts">
import { computed } from "vue";

import { useTimeAgo } from "@vueuse/core";

import { IconAdd, IconRefresh, IconSidebarClose } from "@/icons";

import IconBtn from "@/components/ui/IconBtn.vue";
import Spinner from "@/components/ui/Spinner.vue";

import { useProjectsStore } from "@/stores/projects";
import { useUiStore } from "@/stores/ui";

import ProjectTree from "./ProjectTree.vue";

const projects = useProjectsStore();
const ui = useUiStore();

const lastScan = computed(() => projects.lastScan ?? new Date());
const ago = useTimeAgo(lastScan);
</script>

<template>
  <aside
    class="flex w-[284px] shrink-0 flex-col border-r border-line bg-sidebar"
    aria-label="Projects"
  >
    <div
      class="flex h-11 shrink-0 items-center justify-between border-b border-line pr-2.5 pl-4"
    >
      <span class="font-semibold tracking-tight">Projects</span>
      <span class="flex gap-0.5">
        <IconBtn
          label="Rescan roots"
          size="sm"
          :disabled="projects.loading"
          @click="projects.rescan()"
        >
          <Spinner v-if="projects.loading" class="size-4" />
          <IconRefresh v-else class="size-4" />
        </IconBtn>
        <RouterLink
          to="/roots"
          aria-label="Add root directory"
          title="Add root directory"
          class="grid size-[26px] place-items-center rounded-md text-muted hover:bg-hover hover:text-fg"
        >
          <IconAdd class="size-4" />
        </RouterLink>
        <IconBtn
          label="Collapse sidebar"
          size="sm"
          @click="ui.sidebarCollapsed = true"
        >
          <IconSidebarClose class="size-4" />
        </IconBtn>
      </span>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto">
      <p
        v-if="projects.error"
        class="m-3 rounded-md border border-danger-line bg-danger-bg p-2 text-[12px] text-err-fg"
      >
        {{ projects.error }}
      </p>
      <div
        v-else-if="!projects.loaded"
        class="flex items-center gap-2 p-4 text-muted"
      >
        <Spinner class="size-4" /> Loading…
      </div>
      <p
        v-else-if="!projects.roots.length && !projects.projects.length"
        class="p-4 text-muted"
      >
        No root directories yet.
        <RouterLink to="/roots" class="text-accent-text underline"
          >Add one</RouterLink
        >
        to find your Compose projects.
      </p>
      <ProjectTree v-else />
    </div>

    <RouterLink
      to="/roots"
      class="m-3 flex flex-col gap-0.5 rounded-lg border border-line px-3 py-2.5 hover:bg-hover"
    >
      <span class="text-[12px] font-medium">
        {{ projects.roots.length }}
        {{ projects.roots.length === 1 ? "root" : "roots" }} ·
        {{ projects.projects.length }}
        {{ projects.projects.length === 1 ? "project" : "projects" }}
      </span>
      <span class="text-[11.5px] text-muted">{{
        projects.lastScan ? `Last scan ${ago}` : "Scanned at startup"
      }}</span>
    </RouterLink>
  </aside>
</template>
