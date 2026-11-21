<script setup lang="ts">
import { IconFolderAdd, IconProjects } from "@/icons";

import Btn from "@/components/ui/Btn.vue";
import EmptyState from "@/components/ui/EmptyState.vue";

import { useProjectsStore } from "@/stores/projects";

const projects = useProjectsStore();
</script>

<template>
  <EmptyState
    v-if="projects.loaded && !projects.roots.length"
    title="Point DockDuck at your code"
    text="Add a folder like ~/Code and every Compose project inside it shows up here."
  >
    <template #icon><IconFolderAdd class="size-5" /></template>
    <RouterLink to="/roots"
      ><Btn variant="primary">Add folder…</Btn></RouterLink
    >
  </EmptyState>
  <EmptyState
    v-else
    title="Pick a project"
    text="Choose a project in the sidebar to see its services."
  >
    <template #icon><IconProjects class="size-5" /></template>
  </EmptyState>
</template>
