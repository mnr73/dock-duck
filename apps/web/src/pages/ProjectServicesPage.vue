<script setup lang="ts">
import { computed, ref } from "vue";

import { useRoute, useRouter } from "vue-router";

import { IconService } from "@/icons";

import EmptyState from "@/components/ui/EmptyState.vue";
import Spinner from "@/components/ui/Spinner.vue";

import BuildDialog from "@/features/projects/BuildDialog.vue";
import ServicesTable from "@/features/projects/ServicesTable.vue";

import type { ProjectDetail, ProjectSummary } from "@/api/types";
import { useProjectUsage } from "@/composables/useProjectUsage";
import { useJobsStore } from "@/stores/jobs";

const props = defineProps<{
  project: ProjectSummary;
  detail?: ProjectDetail;
}>();
const emit = defineEmits<{ inspect: [service: string] }>();

const route = useRoute();
const router = useRouter();
const jobs = useJobsStore();

const projectId = computed(() => props.project.id);
const services = computed(() => props.detail?.services);
const { byService } = useProjectUsage(projectId, services);

const busy = computed(() => {
  const label = jobs.running[props.project.id];
  if (!label) return [];
  const j = jobs.current;
  return j?.services?.length
    ? j.services
    : (props.detail?.services.map((s) => s.name) ?? []);
});
const highlight = computed(
  () => (route.query.service as string | undefined) ?? "",
);

function act(service: string, action: "start" | "stop" | "restart" | "pull") {
  jobs.run({
    projectId: props.project.id,
    projectName: props.project.name,
    action,
    services: [service],
  });
}

// one service at a time: the dialog collects the build options first
const buildService = ref<string | null>(null);
const buildOpen = computed({
  get: () => buildService.value !== null,
  set: (v) => {
    if (!v) buildService.value = null;
  },
});
function askBuild(service: string) {
  buildService.value = service;
}
function confirmBuild(o: { noCache: boolean; pull: boolean }) {
  const service = buildService.value;
  if (!service) return;
  jobs.run({
    projectId: props.project.id,
    projectName: props.project.name,
    action: "build",
    services: [service],
    ...o,
  });
}

function toLogs(service: string) {
  router.push({
    path: `/projects/${props.project.id}/logs`,
    query: { service },
  });
}
function toExec(service: string) {
  router.push({
    path: `/projects/${props.project.id}/terminal`,
    query: { service },
  });
}
</script>

<template>
  <div v-if="!detail" class="flex items-center gap-2 p-5 text-muted">
    <Spinner class="size-4" /> Loading services…
  </div>
  <EmptyState
    v-else-if="!detail.services.length"
    title="No services found"
    text="This project’s Compose file declares no services, or it could not be read."
  >
    <template #icon><IconService class="size-5" /></template>
  </EmptyState>
  <ServicesTable
    v-else
    :services="detail.services"
    :usage="byService"
    :busy="busy"
    :highlight="highlight"
    @action="act"
    @build="askBuild"
    @inspect="emit('inspect', $event)"
    @logs="toLogs"
    @exec="toExec"
  />
  <BuildDialog
    v-model:open="buildOpen"
    :project-name="project.name"
    :services="buildService ? [buildService] : []"
    @confirm="confirmBuild"
  />
</template>
