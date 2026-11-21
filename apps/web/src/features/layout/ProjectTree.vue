<script setup lang="ts">
import { computed, watch } from "vue";

import { useRoute } from "vue-router";

import {
  IconChevronDown,
  IconChevronRight,
  IconContainer,
  IconService,
} from "@/icons";

import StatusDot from "@/components/ui/StatusDot.vue";

import { useProjectsStore } from "@/stores/projects";
import { useSystemStore } from "@/stores/system";
import { useUiStore } from "@/stores/ui";
import {
  containerTone,
  projectTone,
  serviceTone,
  toneText,
} from "@/utils/status";

const projects = useProjectsStore();
const ui = useUiStore();
const system = useSystemStore();
const route = useRoute();

const currentId = computed(() => (route.params.id as string | undefined) ?? "");

function toggle(id: string) {
  ui.toggleProject(id);
}

// expanded projects (also those restored from localStorage) need their services
watch(
  [() => ui.expandedProjects, () => projects.projects],
  () => {
    for (const p of projects.projects) {
      if (
        ui.expandedProjects.includes(p.id) &&
        !projects.details[p.id] &&
        !projects.detailLoading[p.id]
      ) {
        projects.loadDetail(p.id).catch(() => {});
      }
    }
  },
  { immediate: true, deep: true },
);
</script>

<template>
  <div class="flex flex-col px-2 pt-1 pb-2.5 text-[12.5px]">
    <template v-for="g in projects.groups" :key="g.key">
      <div
        class="truncate px-2 pt-2.5 pb-1 text-[11px] text-muted"
        :class="g.root && 'font-mono'"
        :title="g.root"
      >
        {{ g.root ? system.short(g.root) : "Outside your roots" }}
      </div>
      <p v-if="!g.projects.length" class="px-2 py-1 text-[12px] text-dim">
        No Compose projects found.
      </p>

      <template v-for="p in g.projects" :key="p.id">
        <div class="flex items-center">
          <button
            type="button"
            :aria-expanded="ui.expandedProjects.includes(p.id)"
            :aria-label="`${ui.expandedProjects.includes(p.id) ? 'Collapse' : 'Expand'} ${p.name}`"
            class="grid size-5 shrink-0 place-items-center rounded text-muted hover:text-fg"
            @click="toggle(p.id)"
          >
            <component
              :is="
                ui.expandedProjects.includes(p.id)
                  ? IconChevronDown
                  : IconChevronRight
              "
              class="size-3.5"
            />
          </button>
          <RouterLink
            :to="`/projects/${p.id}`"
            class="flex min-w-0 flex-1 items-center gap-2 rounded-md px-1.5 py-1 hover:bg-hover"
            :class="
              currentId === p.id
                ? 'bg-hover text-fg'
                : p.status === 'stopped'
                  ? 'text-muted'
                  : 'text-fg'
            "
          >
            <span
              class="min-w-0 flex-1 truncate"
              :class="currentId === p.id && 'font-medium'"
            >
              {{ p.name }}
              <span
                v-if="projects.hint(p)"
                class="font-mono text-[10.5px] text-dim"
                >· {{ projects.hint(p) }}</span
              >
            </span>
            <span class="font-mono text-[11px] text-muted"
              >{{ p.running }}/{{ p.total }}</span
            >
            <StatusDot :tone="projectTone(p.status)" />
          </RouterLink>
        </div>

        <div
          v-if="ui.expandedProjects.includes(p.id)"
          class="mt-px mb-1 ml-[14px] flex flex-col border-l border-line pl-2"
        >
          <p
            v-if="projects.detailLoading[p.id] && !projects.details[p.id]"
            class="px-2 py-1 text-[12px] text-dim"
          >
            Loading…
          </p>
          <template
            v-for="s in projects.details[p.id]?.services ?? []"
            :key="s.name"
          >
            <div class="flex items-center">
              <button
                v-if="s.containers.length > 1"
                type="button"
                :aria-expanded="
                  ui.expandedServices.includes(`${p.id}/${s.name}`)
                "
                :aria-label="`Toggle containers of ${s.name}`"
                class="grid size-4 shrink-0 place-items-center text-muted hover:text-fg"
                @click="ui.toggleService(`${p.id}/${s.name}`)"
              >
                <component
                  :is="
                    ui.expandedServices.includes(`${p.id}/${s.name}`)
                      ? IconChevronDown
                      : IconChevronRight
                  "
                  class="size-3"
                />
              </button>
              <span v-else class="size-4 shrink-0" />
              <RouterLink
                :to="`/projects/${p.id}/services?service=${s.name}`"
                class="flex min-w-0 flex-1 items-center gap-1.5 rounded px-1.5 py-0.5 text-[12px] text-fg hover:bg-hover"
              >
                <IconService class="size-3 shrink-0 text-dim" />
                <StatusDot :tone="serviceTone(s.status)" size="xs" />
                <span class="min-w-0 flex-1 truncate font-mono">{{
                  s.name
                }}</span>
                <span
                  v-if="s.containers.length > 1"
                  class="font-mono text-[10.5px] text-dim"
                  >×{{ s.containers.length }}</span
                >
              </RouterLink>
            </div>
            <div
              v-if="
                s.containers.length > 1 &&
                ui.expandedServices.includes(`${p.id}/${s.name}`)
              "
              class="mb-0.5 ml-[19px] flex flex-col border-l border-dashed border-line-strong pl-2"
            >
              <div
                v-for="c in s.containers"
                :key="c.id"
                class="flex items-center gap-1.5 px-1.5 py-0.5 font-mono text-[11px] text-muted"
              >
                <IconContainer class="size-3 shrink-0" />
                <span class="min-w-0 flex-1 truncate" :title="c.name">{{
                  c.name
                }}</span>
                <span :class="toneText[containerTone(c)]">{{ c.state }}</span>
              </div>
            </div>
          </template>
          <p
            v-if="
              projects.details[p.id] && !projects.details[p.id].services.length
            "
            class="px-2 py-1 text-[12px] text-dim"
          >
            No services.
          </p>
        </div>
      </template>
    </template>
  </div>
</template>
