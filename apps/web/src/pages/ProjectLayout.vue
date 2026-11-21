<script setup lang="ts">
import { computed, ref, watch } from "vue";

import { useRoute } from "vue-router";

import { toast } from "vue-sonner";

import {
  IconDown,
  IconPlay,
  IconProjects,
  IconRestart,
  IconStop,
} from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import EmptyState from "@/components/ui/EmptyState.vue";

import ProjectHeader from "@/features/projects/ProjectHeader.vue";

import { api, errorMessage } from "@/api/client";
import { useProjectsStore } from "@/stores/projects";

const route = useRoute();
const projects = useProjectsStore();

const id = computed(() => route.params.id as string);
const project = computed(() => projects.byId(id.value));
const detail = computed(() => projects.details[id.value]);
const busy = ref("");

watch(
  [id, () => projects.loaded],
  () => {
    if (projects.loaded && project.value)
      projects.loadDetail(id.value).catch(() => {});
  },
  { immediate: true },
);

async function act(action: string, body: object = {}) {
  busy.value = action;
  try {
    await api.post(`/projects/${id.value}/actions/${action}`, body);
    toast.info(`${action} started`, { description: project.value?.name });
  } catch (e) {
    toast.error(`${action} failed`, { description: errorMessage(e) });
  } finally {
    busy.value = "";
  }
}
</script>

<template>
  <div v-if="project" class="flex min-h-full flex-col">
    <ProjectHeader :project="project">
      <template #actions>
        <Btn
          :variant="project.status === 'running' ? 'secondary' : 'primary'"
          :loading="busy === 'up'"
          @click="act('up')"
        >
          <template #icon><IconPlay class="size-3.5" /></template>
          Up
        </Btn>
        <Btn
          :disabled="project.status === 'running' || !project.total"
          :loading="busy === 'start'"
          @click="act('start')"
        >
          <template #icon><IconPlay class="size-3.5" /></template>
          Start
        </Btn>
        <Btn
          :disabled="project.running === 0"
          :loading="busy === 'stop'"
          @click="act('stop')"
        >
          <template #icon><IconStop class="size-3" /></template>
          Stop
        </Btn>
        <Btn
          :disabled="!project.total"
          :loading="busy === 'restart'"
          @click="act('restart')"
        >
          <template #icon><IconRestart class="size-3.5" /></template>
          Restart
        </Btn>
        <span class="mx-0.5 my-1 w-px bg-line-strong" />
        <Btn
          variant="danger"
          disabled
          title="Down needs the confirmation dialog (Phase 4)"
        >
          <template #icon><IconDown class="size-3.5" /></template>
          Down…
        </Btn>
      </template>
    </ProjectHeader>

    <div v-if="detail?.config_error" class="px-4 pt-3">
      <Alert tone="warn">
        Compose couldn’t resolve this project’s configuration; services come
        from existing containers only.
        <pre
          class="mt-1 font-mono text-[11.5px] whitespace-pre-wrap text-muted"
          >{{ detail.config_error }}</pre>
      </Alert>
    </div>

    <RouterView :project="project" :detail="detail" />
  </div>

  <EmptyState
    v-else-if="projects.loaded"
    title="Project not found"
    text="It may have been removed from your roots, or it belongs to another Docker context."
  >
    <template #icon><IconProjects class="size-5" /></template>
    <RouterLink to="/"><Btn>Back to projects</Btn></RouterLink>
  </EmptyState>
</template>
