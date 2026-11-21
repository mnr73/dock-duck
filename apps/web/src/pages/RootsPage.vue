<script setup lang="ts">
import { computed, ref } from "vue";

import { toast } from "vue-sonner";

import { IconFolder, IconFolderAdd, IconRefresh, IconRemove } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import IconBtn from "@/components/ui/IconBtn.vue";
import StatusDot from "@/components/ui/StatusDot.vue";

import PageHeader from "@/features/layout/PageHeader.vue";

import { errorMessage } from "@/api/client";
import { useProjectsStore } from "@/stores/projects";
import { useSystemStore } from "@/stores/system";
import { projectTone, relativeTo } from "@/utils/status";

const projects = useProjectsStore();
const system = useSystemStore();

const path = ref("");
const adding = ref(false);
const addError = ref("");
const rescanning = ref(false);

const toRemove = ref<{ id: string; path: string } | null>(null);
const confirmOpen = computed({
  get: () => !!toRemove.value,
  set: (v) => {
    if (!v) toRemove.value = null;
  },
});

const IGNORED = [
  "node_modules",
  "vendor",
  "dist",
  "build",
  "target",
  ".venv",
  ".git and other hidden folders",
];

function expand(p: string) {
  p = p.trim();
  if (p === "~" || p.startsWith("~/")) return system.home + p.slice(1);
  return p;
}

async function add() {
  addError.value = "";
  const p = expand(path.value);
  if (!p) return;
  adding.value = true;
  try {
    await projects.addRoot(p);
    toast.success("Root added", { description: system.short(p) });
    path.value = "";
  } catch (e) {
    addError.value = errorMessage(e);
  } finally {
    adding.value = false;
  }
}

async function rescan() {
  rescanning.value = true;
  try {
    await projects.rescan();
    toast.success(`Found ${projects.projects.length} projects`);
  } catch (e) {
    toast.error("Rescan failed", { description: errorMessage(e) });
  } finally {
    rescanning.value = false;
  }
}

async function remove() {
  const r = toRemove.value;
  if (!r) return;
  try {
    await projects.removeRoot(r.id);
    toast.success("Root removed", { description: system.short(r.path) });
  } catch (e) {
    toast.error("Couldn’t remove root", { description: errorMessage(e) });
  }
  toRemove.value = null;
}

const countFor = (rootPath: string) =>
  projects.projects.filter((p) => p.root === rootPath).length;
</script>

<template>
  <div class="flex min-h-full flex-col">
    <PageHeader
      title="Project roots"
      subtitle="DockDuck scans these folders for Compose projects."
    >
      <template #actions>
        <Btn :loading="rescanning" @click="rescan">
          <template #icon><IconRefresh class="size-3.5" /></template>
          Rescan all
        </Btn>
      </template>
    </PageHeader>

    <div class="flex flex-wrap items-start gap-6 p-5">
      <section
        class="flex min-w-[320px] max-w-[560px] flex-1 flex-col gap-3"
        aria-label="Root directories"
      >
        <h2 class="text-[12px] font-medium text-muted">Root directories</h2>

        <div
          class="overflow-hidden rounded-[9px] border border-line bg-sidebar"
        >
          <EmptyState
            v-if="projects.loaded && !projects.roots.length"
            title="Point DockDuck at your code"
            text="Add a folder like ~/Code and every Compose project inside it shows up here."
          >
            <template #icon><IconFolder class="size-5" /></template>
          </EmptyState>

          <div
            v-for="r in projects.roots"
            :key="r.id"
            class="flex items-center gap-3 border-b border-line/70 px-3 py-2.5"
          >
            <IconFolder class="size-4 shrink-0 text-muted" />
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <span class="truncate font-mono" :title="r.path">{{
                system.short(r.path)
              }}</span>
              <span class="text-[11.5px] text-muted">
                {{ countFor(r.path) }}
                {{ countFor(r.path) === 1 ? "project" : "projects" }}
              </span>
            </div>
            <IconBtn :label="`Remove ${r.path}`" @click="toRemove = r"
              ><IconRemove class="size-4"
            /></IconBtn>
          </div>

          <form
            class="flex flex-wrap items-center gap-2 bg-app px-3 py-2.5"
            @submit.prevent="add"
          >
            <label
              class="flex h-[30px] min-w-[200px] flex-1 items-center gap-2 rounded-md border border-line-strong bg-surface px-2.5"
            >
              <IconFolderAdd class="size-4 text-muted" />
              <input
                v-model="path"
                aria-label="New root directory"
                placeholder="~/Sites or /srv/stacks"
                spellcheck="false"
                autocomplete="off"
                class="min-w-0 flex-1 bg-transparent font-mono outline-none placeholder:text-dim"
              />
            </label>
            <Btn
              type="submit"
              variant="primary"
              :loading="adding"
              :disabled="!path.trim()"
              >Add &amp; scan</Btn
            >
          </form>
        </div>
        <Alert v-if="addError" tone="err">{{ addError }}</Alert>

        <div class="flex flex-col gap-1.5 rounded-[9px] border border-line p-3">
          <span class="text-muted">Ignored while scanning</span>
          <span class="flex flex-wrap gap-1.5 font-mono text-[11.5px]">
            <span
              v-for="i in IGNORED"
              :key="i"
              class="rounded bg-raised px-1.5 py-0.5"
              >{{ i }}</span
            >
          </span>
        </div>
      </section>

      <section
        class="flex min-w-[320px] flex-[2_1_420px] flex-col gap-3"
        aria-label="Discovered projects"
      >
        <h2 class="text-[12px] font-medium text-muted">
          Discovered projects · {{ projects.projects.length }}
        </h2>
        <div class="overflow-hidden rounded-[9px] border border-line">
          <p v-if="!projects.projects.length" class="p-4 text-muted">
            Nothing discovered yet.
          </p>
          <RouterLink
            v-for="p in projects.projects"
            :key="p.id"
            :to="`/projects/${p.id}`"
            class="flex items-center gap-3 border-b border-line/70 px-3 py-2 last:border-b-0 hover:bg-hover"
          >
            <StatusDot :tone="projectTone(p.status)" />
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium">{{ p.name }}</span>
              <span
                class="block truncate font-mono text-[11.5px] text-muted"
                :title="p.dir"
              >
                {{ p.root ? relativeTo(p.dir, p.root) : system.short(p.dir) }}
              </span>
            </span>
            <span class="font-mono text-[11px] text-muted"
              >{{ p.running }}/{{ p.total }}</span
            >
          </RouterLink>
        </div>
      </section>
    </div>

    <Dialog v-model:open="confirmOpen" title="Remove this root?" width="440px">
      <p>
        Stop scanning
        <code class="font-mono">{{
          toRemove ? system.short(toRemove.path) : ""
        }}</code>? Its projects disappear from DockDuck. Nothing is deleted from
        disk and running containers keep running.
      </p>
      <template #footer>
        <Btn @click="toRemove = null">Cancel</Btn>
        <Btn variant="danger" @click="remove">Remove root</Btn>
      </template>
    </Dialog>
  </div>
</template>
