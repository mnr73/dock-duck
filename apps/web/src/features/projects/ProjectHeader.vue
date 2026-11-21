<script setup lang="ts">
import { computed } from "vue";

import Tabs, { type TabItem } from "@/components/ui/Tabs.vue";

import type { ProjectSummary } from "@/api/types";
import { useSystemStore } from "@/stores/system";
import { basename, projectLabel, projectTone } from "@/utils/status";

import StatusLabel from "./StatusLabel.vue";

const props = defineProps<{ project: ProjectSummary }>();
const system = useSystemStore();

const tabs = computed<TabItem[]>(() => {
  const base = `/projects/${props.project.id}`;
  return [
    { key: "overview", label: "Overview", to: base },
    { key: "services", label: "Services", to: `${base}/services` },
    { key: "compose", label: "Compose", to: `${base}/compose` },
    { key: "env", label: "Environment", to: `${base}/env` },
    { key: "logs", label: "Logs", to: `${base}/logs` },
    { key: "terminal", label: "Terminal", to: `${base}/terminal` },
  ];
});
</script>

<template>
  <div class="shrink-0 border-b border-line bg-app">
    <header
      class="flex min-h-11 flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2"
    >
      <div class="flex min-w-0 flex-wrap items-baseline gap-x-2.5">
        <h1 class="text-[15px] font-semibold tracking-tight">
          {{ project.name }}
        </h1>
        <span
          class="truncate font-mono text-[11.5px] text-muted"
          :title="project.dir"
        >
          {{ system.short(project.dir) }} ·
          {{ project.files.map(basename).join(", ") }}
        </span>
      </div>
      <StatusLabel :tone="projectTone(project.status)">
        {{ projectLabel(project.status, project.running, project.total) }}
      </StatusLabel>
      <div class="ml-auto flex flex-wrap gap-1.5"><slot name="actions" /></div>
    </header>
    <Tabs :items="tabs" aria-label="Project sections" class="!border-b-0" />
  </div>
</template>
