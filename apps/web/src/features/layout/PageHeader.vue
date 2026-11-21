<script setup lang="ts">
defineProps<{ title: string; subtitle?: string }>();
</script>

<template>
  <header
    class="flex min-h-11 shrink-0 flex-wrap items-center gap-x-3 gap-y-2 border-b border-line px-4 py-2"
  >
    <h1 class="text-[15px] font-semibold tracking-tight">{{ title }}</h1>
    <span v-if="subtitle" class="text-[12px] text-muted">{{ subtitle }}</span>
    <div class="ml-auto flex gap-1.5"><slot name="actions" /></div>
  </header>
</template>
