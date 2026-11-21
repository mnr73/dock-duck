<script setup lang="ts">
import { ref, watch } from "vue";

const props = defineProps<{ value: number; max: number; id: string }>();
const emit = defineEmits<{ commit: [depth: number] }>();

/** edited text; committed on blur/Enter so typing "12" is not saved as "1" then "12" */
const text = ref(String(props.value));
watch(
  () => props.value,
  (v) => (text.value = String(v)),
);

function commit() {
  const n = Number.parseInt(text.value, 10);
  if (!Number.isInteger(n) || n < 1 || n > props.max) {
    text.value = String(props.value); // out of range: put back what is stored
    return;
  }
  if (n !== props.value) emit("commit", n);
}
</script>

<template>
  <input
    :id="id"
    v-model="text"
    type="number"
    inputmode="numeric"
    :min="1"
    :max="max"
    class="h-7 w-16 rounded-md border border-line-strong bg-app px-2 text-center text-[13px] text-fg"
    @blur="commit"
    @keydown.enter.prevent="commit"
  />
</template>
