<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue";

import { onBeforeRouteLeave } from "vue-router";

import { useEventListener } from "@vueuse/core";
import { toast } from "vue-sonner";

import {
  IconAdd,
  IconHide,
  IconRaw,
  IconRemove,
  IconReveal,
  IconSave,
  IconSecret,
  IconTable,
  IconUndo,
} from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Badge from "@/components/ui/Badge.vue";
import Btn from "@/components/ui/Btn.vue";
import SegmentedControl from "@/components/ui/SegmentedControl.vue";
import Spinner from "@/components/ui/Spinner.vue";

import ApplyHint from "@/features/files/ApplyHint.vue";
import CodeEditor from "@/features/files/CodeEditor.vue";

import { ApiError, errorMessage } from "@/api/client";
import { filesApi } from "@/api/files";
import type { EnvView, ProjectDetail, ProjectSummary } from "@/api/types";
import {
  type EnvRow,
  MASK,
  countEnvChanges,
  newRow,
  parseEnv,
  rowsToText,
  toRows,
  validateRows,
} from "@/utils/env";

const props = defineProps<{
  project: ProjectSummary;
  detail?: ProjectDetail;
}>();
defineEmits<{ inspect: [service: string] }>();

const view = shallowRef<EnvView>();
const rows = ref<EnvRow[]>([]);
const raw = ref("");
const mode = ref<"table" | "raw">("table");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const conflict = ref(false);
const revealed = ref(false);
const revealing = ref(false);
/** what is on disk (as last loaded; masked secrets stay masked) */
const baseline = ref("");
const savedAt = ref<Date>();
const applyHint = ref(false);

async function load(reveal = false) {
  loading.value = true;
  error.value = "";
  conflict.value = false;
  try {
    const v = await filesApi.env(props.project.id, reveal);
    view.value = v;
    rows.value = toRows(v.lines);
    revealed.value = v.revealed;
    // the raw view is fetched on demand; keep it consistent with the table
    raw.value = rowsToText(rows.value);
    baseline.value = raw.value;
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    loading.value = false;
  }
}
watch(
  () => props.project.id,
  () => load(),
  { immediate: true },
);

const missing = computed(() => view.value?.example?.missing ?? []);
const interpolated = computed(
  () => new Set(view.value?.usage.interpolated ?? []),
);
const undefinedVars = computed(() => {
  const have = new Set(
    rows.value.filter((r) => r.kind === "var").map((r) => r.key),
  );
  return [...interpolated.value].filter((k) => !have.has(k));
});
const exists = computed(() => view.value?.file.exists ?? false);

/** Secrets are masked until the user reveals them: only then can the file be saved safely. */
const hasMaskedSecrets = computed(() => rows.value.some((r) => r.masked));
const text = computed(() =>
  mode.value === "table" ? rowsToText(rows.value) : raw.value,
);
const dirty = computed(
  () => !loading.value && !!view.value && text.value !== baseline.value,
);
const changes = computed(() => countEnvChanges(baseline.value, text.value));
const rowProblems = computed(
  () => new Map(validateRows(rows.value).map((p) => [p.row, p.message])),
);
const canSave = computed(
  () =>
    dirty.value &&
    !saving.value &&
    rowProblems.value.size === 0 &&
    (mode.value === "raw"
      ? !raw.value.includes(MASK)
      : !hasMaskedSecrets.value || revealed.value),
);

// keep the two views in sync when switching
watch(mode, (m) => {
  if (m === "raw") raw.value = rowsToText(rows.value);
  else rows.value = parseEnv(raw.value);
});

async function reveal() {
  if (
    dirty.value &&
    !window.confirm(
      "Revealing reloads the file and discards your unsaved edits. Continue?",
    )
  )
    return;
  revealing.value = true;
  await load(true);
  revealing.value = false;
}

async function hide() {
  if (
    dirty.value &&
    !window.confirm(
      "Hiding reloads the file and discards your unsaved edits. Continue?",
    )
  )
    return;
  await load(false);
}

function addRow(key = "", value = "") {
  rows.value.push(newRow(key, value));
}

function removeRow(id: number) {
  rows.value = rows.value.filter((r) => r.id !== id);
}

function addMissing() {
  for (const m of missing.value) rows.value.push(newRow(m.key, m.value));
  mode.value = "table";
}

function discard() {
  rows.value = parseEnv(baseline.value);
  raw.value = baseline.value;
  error.value = "";
}

async function save() {
  if (!canSave.value || !view.value) return;
  saving.value = true;
  error.value = "";
  try {
    const body = text.value;
    await filesApi.save(props.project.id, "env", body, view.value.etag);
    savedAt.value = new Date();
    toast.success(".env saved");
    applyHint.value = true;
    await load(revealed.value);
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) conflict.value = true;
    else error.value = errorMessage(e);
  } finally {
    saving.value = false;
  }
}

useEventListener(window, "beforeunload", (e) => {
  if (dirty.value) e.preventDefault();
});
onBeforeRouteLeave(() => {
  if (
    dirty.value &&
    !window.confirm("You have unsaved changes. Leave without saving?")
  )
    return false;
});
useEventListener(window, "keydown", (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && e.key === "s") {
    e.preventDefault();
    save();
  }
});

const modeOptions = [
  { value: "table" as const, label: "Table", icon: IconTable },
  { value: "raw" as const, label: "Raw", icon: IconRaw },
];
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div
      class="flex flex-wrap items-center gap-2 border-b border-line px-4 py-2"
    >
      <SegmentedControl
        v-model="mode"
        :options="modeOptions"
        aria-label="View"
      />
      <span class="font-mono text-[12px] text-fg">.env</span>
      <Badge v-if="view && !exists" tone="warn">not created yet</Badge>
      <span
        v-if="view?.mode"
        class="text-[11.5px] text-dim"
        title="File permissions"
        >{{ view.mode }}</span
      >

      <div class="ml-auto flex items-center gap-2">
        <Badge v-if="dirty" tone="accent"
          >{{ changes }} {{ changes === 1 ? "change" : "changes" }}</Badge
        >
        <Badge v-else-if="savedAt" tone="ok">Saved</Badge>
        <Btn v-if="!revealed" size="sm" :loading="revealing" @click="reveal">
          <IconReveal class="size-3.5" /> Reveal secrets
        </Btn>
        <Btn v-else size="sm" @click="hide">
          <IconHide class="size-3.5" /> Hide secrets
        </Btn>
        <Btn size="sm" :disabled="!dirty" @click="discard">
          <IconUndo class="size-3.5" /> Discard
        </Btn>
        <Btn
          size="sm"
          variant="primary"
          :loading="saving"
          :disabled="!canSave"
          title="Save (Ctrl+S)"
          @click="save"
        >
          <IconSave class="size-3.5" /> Save
        </Btn>
      </div>
    </div>

    <ApplyHint v-model="applyHint" :project="project" what=".env" />

    <div v-if="conflict" class="px-4 pt-3">
      <Alert tone="warn">
        <div class="flex items-center gap-3">
          <span class="flex-1"
            >The file was changed on disk after you opened it. Saving now would
            overwrite that change.</span
          >
          <Btn size="sm" @click="load(revealed)"
            >Reload from disk (lose my edits)</Btn
          >
        </div>
      </Alert>
    </div>
    <div v-if="error" class="px-4 pt-3">
      <Alert tone="err">{{ error }}</Alert>
    </div>
    <div v-if="hasMaskedSecrets && dirty" class="px-4 pt-3">
      <Alert tone="info"
        >Secret values are hidden. Reveal them to save, so a hidden value is
        never written over the real one.</Alert
      >
    </div>

    <div
      v-if="loading && !view"
      class="flex flex-1 items-center justify-center text-muted"
    >
      <Spinner class="size-4" />
    </div>

    <!-- raw -->
    <CodeEditor
      v-else-if="mode === 'raw'"
      v-model="raw"
      language="plain"
      placeholder="KEY=value"
      class="min-h-[260px]"
      @save="save"
    />

    <!-- table -->
    <div v-else class="min-h-0 flex-1 overflow-auto">
      <div
        v-if="missing.length"
        class="m-4 flex items-center gap-3 rounded-lg border border-line-strong bg-raised px-3 py-2 text-[12.5px]"
      >
        <span class="flex-1">
          <span class="font-mono">{{ view?.example?.name }}</span> has
          {{ missing.length }}
          {{ missing.length === 1 ? "key" : "keys" }} missing here:
          <span class="font-mono text-muted">{{
            missing.map((m) => m.key).join(", ")
          }}</span>
        </span>
        <Btn size="sm" @click="addMissing"
          ><IconAdd class="size-3.5" /> Add them</Btn
        >
      </div>

      <table class="w-full text-[12.5px]">
        <thead class="sticky top-0 bg-app text-left text-[11.5px] text-dim">
          <tr>
            <th class="w-[34%] px-4 py-1.5 font-normal">Name</th>
            <th class="px-2 py-1.5 font-normal">Value</th>
            <th class="w-28 px-2 py-1.5 font-normal" />
            <th class="w-10" />
          </tr>
        </thead>
        <tbody>
          <template v-for="r in rows" :key="r.id">
            <tr v-if="r.kind === 'var'" class="border-t border-line align-top">
              <td class="px-4 py-1">
                <div class="flex items-center gap-1.5">
                  <span
                    class="flex size-3 shrink-0 items-center justify-center"
                  >
                    <IconSecret
                      v-if="r.secret"
                      class="size-3 text-dim"
                      aria-label="Secret"
                    />
                  </span>
                  <input
                    v-model="r.key"
                    class="w-full rounded bg-transparent px-1 py-0.5 font-mono outline-none focus:bg-raised"
                    :class="rowProblems.has(r.id) && 'text-err-fg'"
                    placeholder="NAME"
                    spellcheck="false"
                    @input="r.touched = true"
                  />
                </div>
                <p
                  v-if="rowProblems.has(r.id)"
                  class="px-1 pb-0.5 text-[11px] text-err-fg"
                >
                  {{ rowProblems.get(r.id) }}
                </p>
              </td>
              <td class="px-2 py-1">
                <div
                  v-if="r.masked"
                  class="flex items-center gap-2 px-1 py-0.5 text-dim"
                >
                  <span class="font-mono tracking-widest">••••••••</span>
                  <span class="text-[11px]">hidden</span>
                </div>
                <input
                  v-else
                  v-model="r.value"
                  class="w-full rounded bg-transparent px-1 py-0.5 font-mono outline-none focus:bg-raised"
                  :type="r.secret && !revealed ? 'password' : 'text'"
                  placeholder="(empty)"
                  spellcheck="false"
                  autocomplete="off"
                  @input="r.touched = true"
                />
              </td>
              <td class="px-2 py-1.5 text-[11.5px]">
                <span
                  v-if="interpolated.has(r.key)"
                  class="rounded bg-raised px-1.5 py-0.5 text-muted"
                  title="Referenced as ${VAR} in the Compose file"
                  >in compose</span
                >
              </td>
              <td class="px-2 py-1 text-right">
                <button
                  type="button"
                  class="rounded p-1 text-dim hover:bg-hover hover:text-err"
                  :aria-label="`Remove ${r.key}`"
                  @click="removeRow(r.id)"
                >
                  <IconRemove class="size-3.5" />
                </button>
              </td>
            </tr>
            <tr v-else-if="r.kind === 'comment'" class="border-t border-line">
              <td colspan="4" class="px-4 py-1 font-mono text-[12px] text-dim">
                {{ r.raw }}
              </td>
            </tr>
            <tr v-else-if="r.kind === 'invalid'" class="border-t border-line">
              <td
                colspan="4"
                class="px-4 py-1 font-mono text-[12px] text-warn-fg"
                title="Not a KEY=value line; kept as is"
              >
                {{ r.raw }}
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="px-4 py-3">
        <Btn size="sm" @click="addRow()"
          ><IconAdd class="size-3.5" /> Add variable</Btn
        >
        <p
          v-if="view?.usage.services.length"
          class="mt-3 text-[11.5px] text-dim"
        >
          Loaded as <span class="font-mono">env_file</span> by
          <span class="font-mono">{{ view.usage.services.join(", ") }}</span>.
        </p>
        <Alert v-if="undefinedVars.length" tone="warn" class="mt-3">
          Compose uses
          <span class="font-mono">{{ undefinedVars.join(", ") }}</span>
          but {{ undefinedVars.length === 1 ? "it is" : "they are" }} not
          defined here; the value stays empty unless set in the shell.
        </Alert>
      </div>
    </div>
  </div>
</template>
