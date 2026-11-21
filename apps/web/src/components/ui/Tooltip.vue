<script setup lang="ts">
defineProps<{ text: string }>();
</script>

<template>
  <span class="group/tip relative inline-flex">
    <slot />
    <span
      role="tooltip"
      class="pointer-events-none absolute top-full left-1/2 z-50 mt-1 -translate-x-1/2 rounded-md border border-line-strong bg-surface px-2 py-1 text-[11.5px] whitespace-nowrap text-fg opacity-0 shadow-pop transition-opacity delay-300 group-hover/tip:opacity-100"
    >
      {{ text }}
    </span>
  </span>
</template>
