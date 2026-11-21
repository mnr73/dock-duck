<script setup lang="ts">
defineProps<{ label: string; hint?: string; for?: string }>();
</script>

<template>
  <div class="flex items-center justify-between gap-4 px-3.5 py-3">
    <div class="min-w-0">
      <label :for="$props.for" class="block text-[13px]">{{ label }}</label>
      <p v-if="hint" class="text-[12px] text-muted">{{ hint }}</p>
    </div>
    <div class="flex shrink-0 items-center gap-2"><slot /></div>
  </div>
</template>
