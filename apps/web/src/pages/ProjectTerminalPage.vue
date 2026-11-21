<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from "vue";

import { useRoute, useRouter } from "vue-router";

import { useStorage } from "@vueuse/core";
import { toast } from "vue-sonner";

import {
  IconAdd,
  IconClose,
  IconCopy,
  IconFontLarger,
  IconFontSmaller,
  IconOpenWindow,
  IconTerminal,
} from "@/icons";

import Btn from "@/components/ui/Btn.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import IconBtn from "@/components/ui/IconBtn.vue";

import TerminalNewDialog from "@/features/terminal/TerminalNewDialog.vue";
import TerminalView from "@/features/terminal/TerminalView.vue";

import { errorMessage } from "@/api/client";
import { terminalsApi } from "@/api/terminals";
import type {
  ProjectDetail,
  ProjectSummary,
  SystemTerminalStatus,
  TerminalOpenRequest,
} from "@/api/types";
import { useTerminalsStore } from "@/stores/terminals";

const props = defineProps<{
  project: ProjectSummary;
  detail?: ProjectDetail;
}>();
defineEmits<{ inspect: [service: string] }>();

const route = useRoute();
const router = useRouter();
const store = useTerminalsStore();

const tabs = computed(() =>
  store.sessions.filter((s) => s.project === props.project.id),
);
const activeId = useStorage<Record<string, string>>(
  "dockduck.terminal.active",
  {},
);
const current = computed(
  () =>
    tabs.value.find((s) => s.id === activeId.value[props.project.id]) ??
    tabs.value[0],
);
const views = useTemplateRef<InstanceType<typeof TerminalView>[]>("views");
const liveState = ref<Record<string, string>>({});

const dialog = ref(false);
const dialogService = ref<string | undefined>();
const opening = ref(false);
const openError = ref("");
const system = ref<SystemTerminalStatus | null>(null);
const services = computed(() => props.detail?.services ?? []);

function select(id: string) {
  activeId.value = { ...activeId.value, [props.project.id]: id };
}

onMounted(async () => {
  await store.refresh();
  terminalsApi
    .system()
    .then((s) => (system.value = s))
    .catch(() => (system.value = null));
  // "Terminal" in a service's row menu arrives as ?service=name: open the dialog for it
  const svc = route.query.service ? String(route.query.service) : "";
  if (svc) {
    dialogService.value = svc;
    dialog.value = true;
    router.replace({ query: {} });
  } else if (!tabs.value.length) {
    dialog.value = true;
  }
});

watch(
  () => props.project.id,
  () => store.refresh(),
);

function openDialog(service?: string) {
  dialogService.value = service;
  openError.value = "";
  dialog.value = true;
}

async function submit(req: TerminalOpenRequest, where: "browser" | "system") {
  opening.value = true;
  openError.value = "";
  try {
    if (where === "system") {
      const r = await terminalsApi.openSystem(props.project.id, req);
      toast.success(`Opened in ${r.terminal}`, { description: r.display });
    } else {
      const info = await store.open(props.project.id, req);
      select(info.id);
    }
    dialog.value = false;
  } catch (e) {
    openError.value = errorMessage(e);
  } finally {
    opening.value = false;
  }
}

async function closeTab(id: string) {
  const s = store.sessions.find((x) => x.id === id);
  const live =
    !["exited", "gone"].includes(liveState.value[id]) && s?.state !== "exited";
  if (
    live &&
    !window.confirm(
      "Close this terminal? The shell and anything it started in the container are stopped.",
    )
  )
    return;
  await store.close(id);
  if (!tabs.value.length) dialog.value = false;
}

function onState(id: string, state: string, code: number | null) {
  liveState.value = { ...liveState.value, [id]: state };
  if (state === "exited") store.markExited(id, code);
}

const tabLabel = (s: (typeof tabs.value)[number]) => {
  const u = s.user ? ` · ${s.user === "0" ? "root" : s.user}` : "";
  return `${s.service}${u}`;
};

const fontSize = computed({
  get: () => store.fontSize,
  set: (v: number) => (store.fontSize = Math.min(24, Math.max(9, v))),
});
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div class="flex items-center gap-1 border-b border-line px-3 py-1.5">
      <div
        class="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto"
        role="tablist"
        aria-label="Terminals"
      >
        <div
          v-for="s in tabs"
          :key="s.id"
          class="group flex shrink-0 items-center gap-1.5 rounded-md border px-2 py-1 text-[12px]"
          :class="
            current?.id === s.id
              ? 'border-line-strong bg-raised text-fg'
              : 'border-transparent text-muted hover:text-fg'
          "
        >
          <button
            type="button"
            role="tab"
            :aria-selected="current?.id === s.id"
            class="flex items-center gap-1.5"
            :title="s.display"
            @click="select(s.id)"
          >
            <span
              class="size-1.5 rounded-full"
              :class="
                s.state === 'exited' ||
                liveState[s.id] === 'exited' ||
                liveState[s.id] === 'gone'
                  ? 'bg-dim'
                  : liveState[s.id] === 'lost'
                    ? 'bg-warn'
                    : 'bg-ok'
              "
            />
            <span class="font-mono">{{ tabLabel(s) }}</span>
          </button>
          <button
            type="button"
            class="rounded p-0.5 text-dim hover:bg-hover hover:text-fg"
            :aria-label="`Close ${tabLabel(s)}`"
            @click="closeTab(s.id)"
          >
            <IconClose class="size-3" />
          </button>
        </div>
        <IconBtn label="New terminal" @click="openDialog()"
          ><IconAdd class="size-3.5"
        /></IconBtn>
      </div>

      <template v-if="current">
        <IconBtn label="Smaller text" @click="fontSize--"
          ><IconFontSmaller class="size-3.5"
        /></IconBtn>
        <span class="w-6 text-center text-[11px] text-dim">{{ fontSize }}</span>
        <IconBtn label="Larger text" @click="fontSize++"
          ><IconFontLarger class="size-3.5"
        /></IconBtn>
        <IconBtn
          label="Copy selection (Ctrl+Shift+C)"
          @click="views?.find((v, i) => tabs[i]?.id === current?.id)?.copy()"
        >
          <IconCopy class="size-3.5" />
        </IconBtn>
      </template>
      <Btn
        v-if="system?.available"
        size="sm"
        :title="`Open a new session in ${system.name}`"
        @click="openDialog()"
      >
        <IconOpenWindow class="size-3.5" /> {{ system.name }}
      </Btn>
    </div>

    <EmptyState
      v-if="!tabs.length"
      title="No terminal open"
      text="Open a shell in a running service. It keeps running if you leave this page."
    >
      <template #icon><IconTerminal class="size-5" /></template>
      <Btn variant="primary" @click="openDialog()"
        ><IconAdd class="size-3.5" /> New terminal</Btn
      >
    </EmptyState>

    <div class="flex min-h-0 flex-1 flex-col">
      <TerminalView
        v-for="s in tabs"
        :id="s.id"
        :key="s.id"
        ref="views"
        :active="current?.id === s.id"
        :font-size="fontSize"
        @state="(st, c) => onState(s.id, st, c)"
        @close="closeTab(s.id)"
      />
    </div>

    <TerminalNewDialog
      v-model:open="dialog"
      :services="services"
      :initial-service="dialogService"
      :system="system"
      :busy="opening"
      :error="openError"
      @submit="submit"
    />
  </div>
</template>
