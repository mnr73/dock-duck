<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";

import { useRoute } from "vue-router";

import {
  IconBuild,
  IconDown,
  IconPlay,
  IconProjects,
  IconPull,
  IconRestart,
  IconStop,
} from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import EmptyState from "@/components/ui/EmptyState.vue";

import BuildDialog from "@/features/projects/BuildDialog.vue";
import DownDialog from "@/features/projects/DownDialog.vue";
import InspectDrawer from "@/features/projects/InspectDrawer.vue";
import JobDrawer from "@/features/projects/JobDrawer.vue";
import ProjectHeader from "@/features/projects/ProjectHeader.vue";

import { useJobsStore } from "@/stores/jobs";
import { usePrefsStore } from "@/stores/prefs";
import { useProjectsStore } from "@/stores/projects";

const route = useRoute();
const projects = useProjectsStore();
const jobs = useJobsStore();

const id = computed(() => route.params.id as string);
const project = computed(() => projects.byId(id.value));
const detail = computed(() => projects.details[id.value]);
const running = computed(() => jobs.running[id.value] ?? "");
const prefs = usePrefsStore();
const downOpen = ref(false);
const buildOpen = ref(false);

// Services unknown yet (detail still loading): keep the buttons usable.
const hasBuildable = computed(
  () => !detail.value || detail.value.services.some((s) => s.buildable),
);
const hasRegistryImages = computed(
  () => !detail.value || detail.value.services.some((s) => !s.buildable),
);
const inspectService = ref<string | null>(null);

watch(
  [id, () => projects.loaded],
  () => {
    if (projects.loaded && project.value)
      projects.loadDetail(id.value).catch(() => {});
    inspectService.value = null;
  },
  { immediate: true },
);

// refresh the page once an operation ends (events usually do this too, this covers a missed one)
jobs.setOnFinished(() => {
  projects.load(true);
  if (projects.details[id.value])
    projects.loadDetail(id.value, true).catch(() => {});
});
onBeforeUnmount(() => jobs.setOnFinished(null));

function act(
  action: string,
  extra: {
    services?: string[];
    volumes?: boolean;
    removeOrphans?: boolean;
    noCache?: boolean;
    pull?: boolean;
  } = {},
) {
  if (!project.value) return;
  jobs.run({
    projectId: id.value,
    projectName: project.value.name,
    action,
    ...extra,
  });
}

function confirmBuild(o: { noCache: boolean; pull: boolean }) {
  act("build", o);
}

/**
 * "Confirm before compose down" can be switched off in Settings: a plain down then
 * runs at once. Removing volumes is irreversible, so that path always goes through
 * the dialog (it is only reachable from there).
 */
function askDown() {
  if (prefs.confirmDown) downOpen.value = true;
  else act("down", { volumes: false, removeOrphans: false });
}

function confirmDown(o: { volumes: boolean; removeOrphans: boolean }) {
  act("down", o);
}
</script>

<template>
  <div v-if="project" class="relative flex h-full min-h-0 flex-col">
    <ProjectHeader :project="project">
      <template #actions>
        <Btn
          :variant="project.status === 'running' ? 'secondary' : 'primary'"
          :loading="running === 'up'"
          :disabled="!!running && running !== 'up'"
          @click="act('up')"
        >
          <template #icon><IconPlay class="size-3.5" /></template>
          Up
        </Btn>
        <Btn
          :disabled="
            project.status === 'running' ||
            !project.total ||
            (!!running && running !== 'start')
          "
          :loading="running === 'start'"
          @click="act('start')"
        >
          <template #icon><IconPlay class="size-3.5" /></template>
          Start
        </Btn>
        <Btn
          :disabled="project.running === 0 || (!!running && running !== 'stop')"
          :loading="running === 'stop'"
          @click="act('stop')"
        >
          <template #icon><IconStop class="size-3" /></template>
          Stop
        </Btn>
        <Btn
          :disabled="!project.total || (!!running && running !== 'restart')"
          :loading="running === 'restart'"
          @click="act('restart')"
        >
          <template #icon><IconRestart class="size-3.5" /></template>
          Restart
        </Btn>
        <span class="mx-0.5 my-1 w-px bg-line-strong" />
        <Btn
          :disabled="!hasRegistryImages || (!!running && running !== 'pull')"
          :loading="running === 'pull'"
          :title="
            hasRegistryImages
              ? 'Pull the newest images of every service that uses a registry image'
              : 'No service uses a registry image (all are built locally)'
          "
          @click="act('pull')"
        >
          <template #icon><IconPull class="size-3.5" /></template>
          Pull
        </Btn>
        <Btn
          :disabled="!hasBuildable || (!!running && running !== 'build')"
          :loading="running === 'build'"
          :title="
            hasBuildable
              ? 'Build the images of every service that has a build section'
              : 'No service has a build section'
          "
          @click="buildOpen = true"
        >
          <template #icon><IconBuild class="size-3.5" /></template>
          Build…
        </Btn>
        <span class="mx-0.5 my-1 w-px bg-line-strong" />
        <Btn
          variant="danger"
          :disabled="!project.total || !!running"
          :loading="running === 'down'"
          @click="askDown"
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

    <div class="flex min-h-0 flex-1 flex-col overflow-auto">
      <RouterView
        :project="project"
        :detail="detail"
        @inspect="inspectService = $event"
      />
    </div>

    <button
      v-if="jobs.current && !jobs.drawerOpen"
      type="button"
      class="absolute right-4 bottom-4 z-20 flex items-center gap-2 rounded-lg border border-line-strong bg-surface px-3 py-1.5 text-[12px] shadow-pop hover:bg-hover"
      @click="jobs.drawerOpen = true"
    >
      <span
        class="size-2 rounded-full"
        :class="
          {
            running: 'animate-pulse bg-warn',
            succeeded: 'bg-ok',
            failed: 'bg-err',
          }[jobs.current.status]
        "
      />
      {{ jobs.projectName }} · {{ jobs.current.action }} — show output
    </button>

    <InspectDrawer
      :project-id="id"
      :service="inspectService"
      @close="inspectService = null"
    />
    <JobDrawer />
    <BuildDialog
      v-model:open="buildOpen"
      :project-name="project.name"
      :services="[]"
      @confirm="confirmBuild"
    />
    <DownDialog
      v-model:open="downOpen"
      :project-id="id"
      :project-name="project.name"
      @confirm="confirmDown"
    />
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
