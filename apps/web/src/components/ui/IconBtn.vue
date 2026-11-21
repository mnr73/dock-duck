<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string;
    size?: "sm" | "md";
    active?: boolean;
    disabled?: boolean;
  }>(),
  { size: "md" },
);
</script>

<template>
  <button
    type="button"
    :aria-label="label"
    :title="label"
    :disabled="disabled"
    class="grid shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-hover hover:text-fg disabled:opacity-45"
    :class="[
      size === 'sm' ? 'size-[26px]' : 'size-7',
      active && 'bg-hover text-fg',
    ]"
  >
    <slot />
  </button>
</template>
