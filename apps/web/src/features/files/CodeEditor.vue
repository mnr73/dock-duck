<script setup lang="ts">
import { useTemplateRef } from "vue";

import type { Problem } from "@/api/types";
import { useCodeEditor } from "@/composables/useCodeEditor";

const props = defineProps<{
  language: "yaml" | "plain";
  readOnly?: boolean;
  placeholder?: string;
}>();
const emit = defineEmits<{ save: [] }>();
const model = defineModel<string>({ required: true });

const host = useTemplateRef<HTMLElement>("host");
const editor = useCodeEditor(host, model, {
  language: props.language,
  readOnly: () => props.readOnly,
  placeholder: props.placeholder,
  onSave: () => emit("save"),
});

defineExpose({
  setProblems: (p: Problem[]) => editor.setProblems(p),
  goToLine: (n: number) => editor.goToLine(n),
  focus: () => editor.focus(),
});
</script>

<template>
  <div ref="host" class="min-h-0 flex-1 overflow-hidden" />
</template>
