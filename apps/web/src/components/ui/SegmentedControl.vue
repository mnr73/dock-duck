<script setup lang="ts" generic="T extends string">
import type { Component } from "vue";

defineProps<{
  options: { value: T; label: string; icon?: Component }[];
  ariaLabel?: string;
}>();
const model = defineModel<T>({ required: true });
</script>

<template>
  <div
    role="radiogroup"
    :aria-label="ariaLabel"
    class="inline-flex rounded-md border border-line-strong bg-surface p-0.5"
  >
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      role="radio"
      :aria-checked="model === o.value"
      class="flex h-6 items-center gap-1.5 rounded px-2.5 text-[12px] transition-colors"
      :class="
        model === o.value ? 'bg-raised text-fg' : 'text-muted hover:text-fg'
      "
      @click="model = o.value"
    >
      <component :is="o.icon" v-if="o.icon" class="size-3.5" />
      {{ o.label }}
    </button>
  </div>
</template>
