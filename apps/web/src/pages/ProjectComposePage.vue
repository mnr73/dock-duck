<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from "vue";

import { useRoute, useRouter } from "vue-router";

import { toast } from "vue-sonner";

import { IconCheckFile, IconSave, IconUndo } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Badge from "@/components/ui/Badge.vue";
import Btn from "@/components/ui/Btn.vue";
import Spinner from "@/components/ui/Spinner.vue";

import ApplyHint from "@/features/files/ApplyHint.vue";
import CodeEditor from "@/features/files/CodeEditor.vue";
import ProblemsList from "@/features/files/ProblemsList.vue";

import type { ProjectDetail, ProjectSummary } from "@/api/types";
import { useFileEditor } from "@/composables/useFileEditor";
import { countLineChanges } from "@/utils/env";

const props = defineProps<{
  project: ProjectSummary;
  detail?: ProjectDetail;
}>();
defineEmits<{ inspect: [service: string] }>();

const route = useRoute();
const router = useRouter();

// which compose file: ?file=1 selects the second file of a multi-file project
const fileIndex = computed(() => {
  const n = Number(route.query.file ?? 0);
  return Number.isInteger(n) && n >= 0 && n < props.project.files.length
    ? n
    : 0;
});
const fileId = computed(() => `compose-${fileIndex.value}`);

const ed = useFileEditor(
  () => props.project.id,
  () => fileId.value,
  {
    validate: true,
  },
);
const editor = useTemplateRef<InstanceType<typeof CodeEditor>>("editor");

const changed = computed(() =>
  ed.loaded.value
    ? countLineChanges(ed.loaded.value.content, ed.text.value)
    : 0,
);

// squiggles follow the latest validation; stale ones (text edited since) are hidden
watch(
  [() => ed.problems.value, () => ed.problemsStale.value],
  ([p, stale]) => editor.value?.setProblems(stale ? [] : p),
  { flush: "post" },
);

const errorCount = computed(
  () => ed.problems.value.filter((p) => p.severity === "error").length,
);
const warningCount = computed(
  () => ed.problems.value.filter((p) => p.severity === "warning").length,
);

const applyHint = ref(false);

const baseName = (p: string) => p.slice(p.lastIndexOf("/") + 1);

async function onSave(force = false) {
  const ok = await ed.save(force);
  if (ok) {
    toast.success(`${ed.loaded.value?.name} saved`);
    applyHint.value = true;
  }
}

function selectFile(i: number) {
  router.replace({ query: i ? { file: String(i) } : {} });
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <!-- toolbar -->
    <div
      class="flex flex-wrap items-center gap-2 border-b border-line px-4 py-2"
    >
      <div
        v-if="project.files.length > 1"
        class="flex gap-1"
        role="tablist"
        aria-label="Compose files"
      >
        <button
          v-for="(f, i) in project.files"
          :key="f"
          type="button"
          role="tab"
          :aria-selected="i === fileIndex"
          class="rounded px-2 py-0.5 font-mono text-[12px]"
          :class="
            i === fileIndex ? 'bg-raised text-fg' : 'text-muted hover:text-fg'
          "
          @click="selectFile(i)"
        >
          {{ baseName(f) }}
        </button>
      </div>
      <span v-else class="font-mono text-[12px] text-fg">{{
        ed.loaded.value?.name ?? baseName(project.files[0] ?? "")
      }}</span>
      <span
        class="hidden truncate text-[11.5px] text-dim md:inline"
        :title="ed.loaded.value?.path"
        >{{ ed.loaded.value?.path }}</span
      >

      <div class="ml-auto flex items-center gap-2">
        <Badge v-if="ed.dirty.value" tone="accent"
          >{{ changed }} {{ changed === 1 ? "line" : "lines" }} changed</Badge
        >
        <Badge v-else-if="ed.savedAt.value" tone="ok">Saved</Badge>
        <Spinner v-if="ed.validating.value" class="size-3.5" />
        <Btn size="sm" :disabled="!ed.dirty.value" @click="ed.discard()">
          <IconUndo class="size-3.5" /> Discard
        </Btn>
        <Btn
          size="sm"
          :loading="ed.validating.value"
          :disabled="ed.loading.value"
          @click="ed.validate()"
        >
          <IconCheckFile class="size-3.5" /> Validate
        </Btn>
        <Btn
          size="sm"
          variant="primary"
          :loading="ed.saving.value"
          :disabled="!ed.dirty.value"
          title="Save (Ctrl+S)"
          @click="onSave()"
        >
          <IconSave class="size-3.5" /> Save
        </Btn>
      </div>
    </div>

    <ApplyHint v-model="applyHint" :project="project" what="Compose file" />

    <div v-if="ed.conflict.value" class="px-4 pt-3">
      <Alert tone="warn">
        <div class="flex items-center gap-3">
          <span class="flex-1"
            >This file was changed on disk after you opened it. Saving now would
            overwrite that change.</span
          >
          <Btn size="sm" @click="ed.load()"
            >Reload from disk (lose my edits)</Btn
          >
        </div>
      </Alert>
    </div>
    <div v-if="ed.error.value" class="px-4 pt-3">
      <Alert tone="err">{{ ed.error.value }}</Alert>
    </div>

    <div
      v-if="ed.loading.value && !ed.loaded.value"
      class="flex flex-1 items-center justify-center text-muted"
    >
      <Spinner class="size-4" />
    </div>
    <CodeEditor
      v-else-if="ed.loaded.value"
      ref="editor"
      v-model="ed.text.value"
      language="yaml"
      class="min-h-[260px]"
      @save="onSave()"
    />

    <!-- problems -->
    <div
      v-if="ed.problems.value.length"
      class="max-h-44 shrink-0 overflow-auto border-t border-line bg-surface"
    >
      <div class="flex items-center gap-2 px-3 py-1.5 text-[11.5px] text-muted">
        <span>
          {{ errorCount }} {{ errorCount === 1 ? "error" : "errors" }},
          {{ warningCount }} {{ warningCount === 1 ? "warning" : "warnings" }}
          <template v-if="ed.problemsStale.value">
            (from before your last edit)</template
          >
        </span>
        <button
          v-if="ed.hasErrors.value && !ed.problemsStale.value"
          type="button"
          class="ml-auto text-[11.5px] text-muted underline hover:text-fg"
          @click="onSave(true)"
        >
          Save anyway
        </button>
      </div>
      <ProblemsList
        :problems="ed.problems.value"
        :stale="ed.problemsStale.value"
        @goto="editor?.goToLine($event)"
      />
    </div>
  </div>
</template>
