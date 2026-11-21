<script setup lang="ts">
import { onMounted, watch } from "vue";

import { IconSidebarOpen } from "@/icons";

import IconBtn from "@/components/ui/IconBtn.vue";

import { useLiveEvents } from "@/composables/useLiveEvents";
import { useContextsStore } from "@/stores/contexts";
import { useProjectsStore } from "@/stores/projects";
import { useSystemStore } from "@/stores/system";
import { useUiStore } from "@/stores/ui";

import AppRail from "./AppRail.vue";
import AppSidebar from "./AppSidebar.vue";
import DockerBanner from "./DockerBanner.vue";
import StatusBar from "./StatusBar.vue";

const ui = useUiStore();
const system = useSystemStore();
const contexts = useContextsStore();
const projects = useProjectsStore();
const { live } = useLiveEvents();

onMounted(async () => {
  await Promise.all([system.load(), contexts.load()]);
  await projects.load();
});

// a different context shows different containers
watch(
  () => contexts.selected?.name,
  (now, before) => {
    if (before && now !== before) {
      projects.resetDetails();
      projects.load();
    }
  },
);
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex min-h-0 flex-1">
      <AppRail />
      <AppSidebar v-if="!ui.sidebarCollapsed" />
      <main class="relative flex min-w-0 flex-1 flex-col bg-app">
        <IconBtn
          v-if="ui.sidebarCollapsed"
          label="Expand sidebar"
          class="absolute top-2 left-2 z-10"
          size="sm"
          @click="ui.sidebarCollapsed = false"
        >
          <IconSidebarOpen class="size-4" />
        </IconBtn>
        <DockerBanner />
        <div class="min-h-0 flex-1 overflow-auto"><RouterView /></div>
      </main>
    </div>
    <StatusBar :live="live" />
  </div>
</template>
