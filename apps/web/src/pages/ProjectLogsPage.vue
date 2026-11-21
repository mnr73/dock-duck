<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from "vue";

import { useRoute, useRouter } from "vue-router";

import {
  onKeyStroke,
  refDebounced,
  useClipboard,
  useStorage,
} from "@vueuse/core";
import { toast } from "vue-sonner";

import { IconCopy } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";

import LogToolbar from "@/features/logs/LogToolbar.vue";
import LogView from "@/features/logs/LogView.vue";

import type { ProjectDetail, ProjectSummary } from "@/api/types";
import { useLogStream } from "@/composables/useLogStream";
import { useContextsStore } from "@/stores/contexts";
import {
  type LevelFilter,
  buildMatcher,
  formatLine,
  matches,
  passesLevel,
} from "@/utils/logs";

const props = defineProps<{
  project: ProjectSummary;
  detail?: ProjectDetail;
}>();

const route = useRoute();
const router = useRouter();
const contexts = useContextsStore();

// ---- state (preferences persisted, selections per visit) ----
const services = ref<string[]>(
  route.query.service ? String(route.query.service).split(",") : [],
);
const since = ref("");
const level = ref<LevelFilter>("all");
const query = ref("");
const regex = ref(false);
const showTime = useStorage("dockduck.logs.time", true);
const wrap = useStorage("dockduck.logs.wrap", false);
const follow = ref(true);
const selected = ref<Set<number>>(new Set());
const view = useTemplateRef<InstanceType<typeof LogView>>("view");
const toolbar = useTemplateRef<InstanceType<typeof LogToolbar>>("toolbar");

// keep the URL (?service=) in sync so a link/reload reproduces the filter
watch(services, (v) =>
  router.replace({
    query: v.length
      ? { ...route.query, service: v.join(",") }
      : { ...route.query, service: undefined },
  }),
);
watch(
  () => route.query.service,
  (q) => {
    const next = q ? String(q).split(",") : [];
    if (next.join() !== services.value.join()) services.value = next;
  },
);

const stream = useLogStream({
  projectId: () => props.project.id,
  services,
  tail: 500,
  since,
  context: () => contexts.selected?.name,
});

// ---- filtering ----
const debouncedQuery = refDebounced(query, 150);
const matcher = computed(() => buildMatcher(debouncedQuery.value, regex.value));
const shown = computed(() => {
  const m = matcher.value;
  const hasQuery = !!m.re;
  if (!hasQuery && level.value === "all") return stream.lines.value;
  return stream.lines.value.filter(
    (l) => passesLevel(l.level, level.value) && (!hasQuery || matches(l.m, m)),
  );
});
const serviceNames = computed(
  () => props.detail?.services.map((s) => s.name) ?? [],
);
const showService = computed(
  () => serviceNames.value.length > 1 && services.value.length !== 1,
);

// ---- copy / export ----
const { copy, copied } = useClipboard({ copiedDuring: 1500 });
const width = computed(() =>
  Math.min(
    24,
    shown.value.reduce((n, l) => Math.max(n, l.s.length), 0),
  ),
);

const selectedLines = computed(() =>
  shown.value.filter((l) => selected.value.has(l.id)),
);
async function copySelected() {
  await copy(
    selectedLines.value
      .map((l) => formatLine(l, showTime.value, width.value))
      .join("\n"),
  );
  toast.success(`Copied ${selectedLines.value.length} lines`);
}

function download() {
  const q = new URLSearchParams();
  if (services.value.length) q.set("services", services.value.join(","));
  if (since.value) q.set("since", since.value);
  const ctx = contexts.selected?.name;
  if (ctx) q.set("context", ctx);
  // a normal navigation to an attachment endpoint: the browser handles the file save
  const a = document.createElement("a");
  a.href = `/api/v1/projects/${props.project.id}/logs/download?${q}`;
  a.download = "";
  document.body.appendChild(a);
  a.click();
  a.remove();
}

function clear() {
  stream.clear();
  selected.value = new Set();
}

function pause() {
  stream.pause();
  follow.value = false;
}
function resume() {
  stream.resume();
  follow.value = true;
}

// Ctrl/Cmd+F focuses the log search instead of the browser's
onKeyStroke("f", (e) => {
  if (e.ctrlKey || e.metaKey) {
    e.preventDefault();
    toolbar.value?.focusSearch();
  }
});
onKeyStroke("Escape", () => {
  if (selected.value.size) selected.value = new Set();
});

const noContainers = computed(
  () =>
    !!props.detail &&
    props.detail.services.every((s) => s.containers.length === 0),
);

const emptyText = computed(() => {
  if (noContainers.value)
    return "No containers yet. Start the project to see its logs here.";
  if (stream.ended.value) return "No log output.";
  return stream.live.value ? "Waiting for output…" : "Connecting…";
});
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <LogToolbar
      ref="toolbar"
      v-model:selected-services="services"
      v-model:query="query"
      v-model:regex="regex"
      v-model:level="level"
      v-model:since="since"
      v-model:show-time="showTime"
      v-model:wrap="wrap"
      v-model:follow="follow"
      :paused="stream.paused.value"
      :services="detail?.services ?? []"
      :live="stream.live.value"
      :ended="stream.ended.value"
      :idle="noContainers"
      :visible-count="shown.length"
      :total-count="stream.lines.value.length"
      :search-error="matcher.error"
      @update:paused="(v: boolean) => (v ? pause() : resume())"
      @pause="pause"
      @resume="resume"
      @clear="clear"
      @download="download"
      @restart="stream.restart"
    />

    <Alert
      v-if="stream.ended.value && stream.endError.value"
      tone="warn"
      class="m-3"
    >
      The log stream ended: {{ stream.endError.value }}
    </Alert>

    <LogView
      ref="view"
      v-model:follow="follow"
      v-model:selected="selected"
      :lines="shown"
      :matcher="matcher"
      :show-time="showTime"
      :wrap="wrap"
      :services="serviceNames"
      :show-service="showService"
    >
      <div
        v-if="!stream.lines.value.length"
        class="pointer-events-none absolute inset-0 grid place-items-center text-[12px] text-[#6e6e6e]"
      >
        {{ emptyText }}
      </div>
      <div
        v-else-if="!shown.length"
        class="pointer-events-none absolute inset-0 grid place-items-center text-[12px] text-[#6e6e6e]"
      >
        No lines match the filter.
      </div>
    </LogView>

    <footer
      class="flex h-8 shrink-0 items-center gap-3 border-t border-term-line bg-[#0a0a0a] px-3 text-[11.5px] text-[#a1a1a1]"
    >
      <template v-if="stream.paused.value">
        <span class="text-[#f2c672]"
          >Paused · {{ stream.buffered.value }} new lines buffered</span
        >
        <Btn size="sm" variant="ghost" class="!text-[#f2c672]" @click="resume"
          >Resume</Btn
        >
      </template>
      <template v-else-if="selected.size">
        <span>{{ selected.size }} lines selected</span>
        <Btn
          size="sm"
          variant="ghost"
          class="!text-[#d4d4d4]"
          @click="copySelected"
        >
          <template #icon><IconCopy class="size-3.5" /></template>
          {{ copied ? "Copied" : "Copy" }}
        </Btn>
        <Btn
          size="sm"
          variant="ghost"
          class="!text-[#d4d4d4]"
          @click="selected = new Set()"
          >Clear selection</Btn
        >
      </template>
      <span v-else-if="!stream.lines.value.length">0 lines</span>
      <span v-else
        >{{ shown.length
        }}{{
          shown.length !== stream.lines.value.length
            ? ` of ${stream.lines.value.length}`
            : ""
        }}
        lines · click to select, shift-click for a range</span
      >
    </footer>
  </div>
</template>
