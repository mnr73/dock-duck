<script setup lang="ts">
defineProps<{ title: string; text?: string }>();
</script>

<template>
  <div
    class="mx-auto flex max-w-md flex-col items-center gap-3 px-6 py-14 text-center"
  >
    <div
      v-if="$slots.icon"
      class="grid size-12 place-items-center rounded-xl border border-line-strong bg-raised text-muted"
    >
      <slot name="icon" />
    </div>
    <h2 class="text-[15px] font-semibold">{{ title }}</h2>
    <p v-if="text" class="text-muted">{{ text }}</p>
    <slot />
  </div>
</template>
