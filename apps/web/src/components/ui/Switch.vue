<script setup lang="ts">
defineProps<{ label: string; disabled?: boolean }>();
const model = defineModel<boolean>({ required: true });
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="model"
    :aria-label="label"
    :disabled="disabled"
    class="relative h-5 w-9 shrink-0 rounded-full border border-line-strong transition-colors disabled:opacity-50"
    :class="model ? 'bg-accent' : 'bg-raised'"
    @click="model = !model"
  >
    <span
      class="absolute top-0.5 left-0.5 size-3.5 rounded-full transition-transform"
      :class="model ? 'translate-x-4 bg-accent-fg' : 'bg-muted'"
    />
  </button>
</template>
