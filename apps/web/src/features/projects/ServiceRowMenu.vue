<script setup lang="ts">
import { computed } from "vue";

import {
  IconBuild,
  IconInspect,
  IconLogs,
  IconMore,
  IconPlay,
  IconPull,
  IconRestart,
  IconStop,
  IconTerminal,
} from "@/icons";

import IconBtn from "@/components/ui/IconBtn.vue";
import Menu, { type MenuItem } from "@/components/ui/Menu.vue";

import type { ServiceInfo } from "@/api/types";

const props = defineProps<{ service: ServiceInfo; busy?: boolean }>();
const emit = defineEmits<{
  action: [action: "start" | "stop" | "restart" | "pull"];
  build: [];
  inspect: [];
  logs: [];
  exec: [];
}>();

const anyRunning = computed(() =>
  props.service.containers.some((c) => c.state === "running"),
);
const hasContainers = computed(() => props.service.containers.length > 0);
const buildable = computed(() => props.service.buildable);

const items = computed<MenuItem[]>(() => [
  {
    label: "Start",
    icon: IconPlay,
    disabled: props.busy || anyRunning.value,
    onSelect: () => emit("action", "start"),
  },
  {
    label: "Stop",
    icon: IconStop,
    disabled: props.busy || !anyRunning.value,
    onSelect: () => emit("action", "stop"),
  },
  {
    label: "Restart",
    icon: IconRestart,
    disabled: props.busy || !hasContainers.value,
    onSelect: () => emit("action", "restart"),
  },
  {
    label: "Pull image",
    icon: IconPull,
    separator: true,
    disabled: props.busy || buildable.value,
    hint: buildable.value
      ? "This image is built locally from a build section; use Build instead"
      : "Download the newest version of this service's image",
    onSelect: () => emit("action", "pull"),
  },
  {
    label: "Build image",
    icon: IconBuild,
    disabled: props.busy || !buildable.value,
    hint: buildable.value
      ? "Build this service's image from its build section"
      : "This service has no build section (it uses a prebuilt image)",
    onSelect: () => emit("build"),
  },
  {
    label: "View logs",
    separator: true,
    icon: IconLogs,
    disabled: !hasContainers.value,
    onSelect: () => emit("logs"),
  },
  {
    label: "Execute command…",
    icon: IconTerminal,
    disabled: !anyRunning.value,
    onSelect: () => emit("exec"),
  },
  {
    label: "Inspect",
    icon: IconInspect,
    disabled: !hasContainers.value,
    onSelect: () => emit("inspect"),
  },
]);
</script>

<template>
  <Menu :items="items" align="end">
    <template #trigger="{ toggle }">
      <IconBtn
        :label="`Actions for ${service.name}`"
        size="sm"
        @click.stop="toggle"
      >
        <IconMore class="size-4" />
      </IconBtn>
    </template>
  </Menu>
</template>
