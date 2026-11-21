<script setup lang="ts">
import { computed } from "vue";

import StatusDot from "@/components/ui/StatusDot.vue";

import { useContextsStore } from "@/stores/contexts";
import { useProjectsStore } from "@/stores/projects";
import { useSystemStore } from "@/stores/system";

defineProps<{ live: boolean }>();
defineEmits<{ "open-palette": [] }>();

const contexts = useContextsStore();
const projects = useProjectsStore();
const system = useSystemStore();

const ctx = computed(() => contexts.selected);
</script>

<template>
  <footer
    class="flex h-[26px] shrink-0 items-center gap-4 border-t border-line bg-sidebar px-3 text-[11.5px] text-muted"
  >
    <RouterLink
      to="/contexts"
      class="flex h-5 items-center gap-1.5 rounded bg-raised px-1.5 font-mono text-fg"
      title="Docker context"
    >
      <StatusDot
        :tone="ctx?.available ? 'ok' : contexts.error ? 'err' : 'err'"
        size="xs"
      />
      {{ ctx?.name ?? "no context" }}
    </RouterLink>
    <span v-if="ctx?.version">Engine {{ ctx.version }}</span>
    <span
      >{{ projects.totalContainers }} containers ·
      {{ projects.runningContainers }} running</span
    >
    <button
      type="button"
      class="ml-auto flex h-5 items-center gap-1.5 rounded px-1.5 text-dim hover:bg-hover hover:text-fg"
      title="Command palette (Ctrl+K)"
      @click="$emit('open-palette')"
    >
      <kbd class="font-mono text-[10.5px]">Ctrl K</kbd> commands
    </button>
    <span
      class="flex items-center gap-1.5"
      :title="live ? 'Live updates connected' : 'Live updates disconnected'"
    >
      <StatusDot :tone="live ? 'ok' : 'off'" size="xs" />
      {{ live ? "live" : "offline" }}
    </span>
    <span v-if="system.version" class="font-mono text-dim">{{
      system.version
    }}</span>
  </footer>
</template>
