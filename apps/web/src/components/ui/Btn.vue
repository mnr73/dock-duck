<script setup lang="ts">
import { computed } from "vue";

import { IconSpinner } from "@/icons";

type Variant = "primary" | "secondary" | "ghost" | "danger";

const props = withDefaults(
  defineProps<{
    variant?: Variant;
    size?: "sm" | "md";
    loading?: boolean;
    disabled?: boolean;
    type?: "button" | "submit";
  }>(),
  { variant: "secondary", size: "md", type: "button" },
);

const classes = computed(() => [
  "inline-flex shrink-0 items-center justify-center gap-1.5 rounded-md font-medium whitespace-nowrap transition-colors",
  "disabled:cursor-not-allowed disabled:opacity-45",
  props.size === "sm" ? "h-[26px] px-2 text-[12px]" : "h-7 px-3 text-[12.5px]",
  {
    primary:
      "bg-accent text-accent-fg hover:brightness-110 disabled:bg-raised disabled:text-dim disabled:opacity-100",
    secondary: "border border-line-strong bg-surface text-fg hover:bg-hover",
    ghost: "text-muted hover:bg-hover hover:text-fg",
    danger:
      "border border-danger-line bg-danger-bg text-err-fg hover:brightness-110",
  }[props.variant],
]);
</script>

<template>
  <button :type="type" :class="classes" :disabled="disabled || loading">
    <IconSpinner v-if="loading" class="size-3.5 animate-spin" />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>
