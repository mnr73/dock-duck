<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";

import { useRouter } from "vue-router";

import { useDebounceFn, useStorage } from "@vueuse/core";

import { IconExternal, IconPorts, IconRefresh, IconSearch } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Badge from "@/components/ui/Badge.vue";
import Btn from "@/components/ui/Btn.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import Spinner from "@/components/ui/Spinner.vue";

import PageHeader from "@/features/layout/PageHeader.vue";

import { useContextsStore } from "@/stores/contexts";
import { usePortsStore } from "@/stores/ports";
import { useProjectsStore } from "@/stores/projects";
import { bindLabel, portUrl } from "@/utils/format";

const ports = usePortsStore();
const projects = useProjectsStore();
const contexts = useContextsStore();
const router = useRouter();

const q = ref("");
const exposedOnly = useStorage("dockduck.ports.exposedOnly", false);

onMounted(() => ports.load());
// the project list is refreshed by live events: reuse that signal to refresh ports too
const reload = useDebounceFn(() => ports.load(true), 600);
watch(
  () => projects.projects.map((p) => `${p.id}:${p.running}`).join(),
  reload,
);
watch(
  () => contexts.selected?.name,
  () => ports.load(),
);

const rows = computed(() => {
  const needle = q.value.trim().toLowerCase();
  return ports.list.filter((p) => {
    if (
      exposedOnly.value &&
      p.host_ips.every((ip) => ip === "127.0.0.1" || ip === "::1")
    )
      return false;
    if (!needle) return true;
    return [
      p.host_port,
      p.container_port,
      p.project,
      p.service,
      p.container,
      p.image,
    ]
      .join(" ")
      .toLowerCase()
      .includes(needle);
  });
});

const conflict = (p: { host_port: number; proto: string }) =>
  ports.duplicates.has(`${p.host_port}/${p.proto}`);

function openProject(name?: string) {
  const p = projects.projects.find((x) => x.name === name);
  if (p) router.push(`/projects/${p.id}`);
}
</script>

<template>
  <div class="flex min-h-full flex-col">
    <PageHeader
      title="Ports"
      subtitle="Every host port published by a running container."
    >
      <template #actions>
        <Btn :loading="ports.loading" @click="ports.load()">
          <template #icon><IconRefresh class="size-3.5" /></template>
          Refresh
        </Btn>
      </template>
    </PageHeader>

    <div
      class="flex flex-wrap items-center gap-3 border-b border-line px-4 py-2"
    >
      <label
        class="flex h-[30px] min-w-[220px] flex-1 items-center gap-2 rounded-md border border-line-strong bg-surface px-2.5 sm:max-w-sm"
      >
        <IconSearch class="size-4 text-muted" />
        <input
          v-model="q"
          aria-label="Filter ports"
          placeholder="Filter by port, project, service…"
          class="min-w-0 flex-1 bg-transparent outline-none placeholder:text-dim"
        />
      </label>
      <label class="flex items-center gap-2 text-[12px] text-muted">
        <input v-model="exposedOnly" type="checkbox" /> Reachable from the
        network only
      </label>
      <span class="ml-auto text-[12px] text-muted"
        >{{ rows.length }} of {{ ports.list.length }}</span
      >
    </div>

    <div v-if="ports.error" class="p-4">
      <Alert tone="err">Couldn’t load ports: {{ ports.error }}</Alert>
    </div>
    <div
      v-else-if="!ports.loaded"
      class="flex items-center gap-2 p-5 text-muted"
    >
      <Spinner class="size-4" /> Loading…
    </div>
    <EmptyState
      v-else-if="!ports.list.length"
      title="No published ports"
      text="No running container publishes a host port in this Docker context."
    >
      <template #icon><IconPorts class="size-5" /></template>
    </EmptyState>
    <p v-else-if="!rows.length" class="p-5 text-muted">
      Nothing matches your filter.
    </p>

    <table v-else class="w-full border-collapse text-left">
      <thead>
        <tr class="border-b border-line text-[11.5px] text-muted">
          <th class="py-2 pr-4 pl-4 font-normal">host port</th>
          <th class="py-2 pr-4 font-normal">container port</th>
          <th class="py-2 pr-4 font-normal">bound to</th>
          <th class="py-2 pr-4 font-normal">project / service</th>
          <th class="py-2 pr-4 font-normal">container</th>
          <th class="py-2 pr-4 font-normal">image</th>
          <th class="w-10 py-2 pr-3"></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="p in rows"
          :key="`${p.container_id}-${p.host_port}-${p.proto}`"
          class="border-b border-line/60 hover:bg-hover/40"
        >
          <td class="py-2 pr-4 pl-4 font-mono font-semibold">
            {{ p.host_port
            }}<span class="font-normal text-dim">/{{ p.proto }}</span>
            <Badge v-if="conflict(p)" tone="err" class="ml-1.5"
              >duplicate</Badge
            >
          </td>
          <td class="py-2 pr-4 font-mono text-muted">{{ p.container_port }}</td>
          <td
            class="py-2 pr-4 text-[12px]"
            :class="
              bindLabel(p.host_ips) === 'localhost only'
                ? 'text-muted'
                : 'text-warn-fg'
            "
            :title="p.host_ips.join(', ')"
          >
            {{ bindLabel(p.host_ips) }}
          </td>
          <td class="py-2 pr-4">
            <button
              v-if="p.project"
              type="button"
              class="text-left hover:text-accent-text"
              @click="openProject(p.project)"
            >
              <span class="font-medium">{{ p.project }}</span>
              <span v-if="p.service" class="font-mono text-[12px] text-muted">
                / {{ p.service }}</span
              >
            </button>
            <span v-else class="text-dim">not a Compose container</span>
          </td>
          <td class="py-2 pr-4 font-mono text-[12px] text-muted">
            {{ p.container }}
          </td>
          <td
            class="max-w-[200px] truncate py-2 pr-4 font-mono text-[12px] text-muted"
            :title="p.image"
          >
            {{ p.image }}
          </td>
          <td class="py-2 pr-3 text-right">
            <a
              v-if="p.proto === 'tcp'"
              :href="portUrl(p.host_port, p.container_port)"
              target="_blank"
              rel="noopener"
              :aria-label="`Open port ${p.host_port}`"
              :title="`Open ${portUrl(p.host_port, p.container_port)}`"
              class="grid size-7 place-items-center rounded-md text-muted hover:bg-hover hover:text-fg"
            >
              <IconExternal class="size-4" />
            </a>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
