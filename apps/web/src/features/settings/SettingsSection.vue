<script setup lang="ts">
defineProps<{ title: string; hint?: string }>();
</script>

<template>
  <section class="rounded-[9px] border border-line">
    <header class="border-b border-line px-3.5 py-2.5">
      <h2 class="font-medium">{{ title }}</h2>
      <p v-if="hint" class="text-[12px] text-muted">{{ hint }}</p>
    </header>
    <div class="divide-y divide-line"><slot /></div>
  </section>
</template>
