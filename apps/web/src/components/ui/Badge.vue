<script setup lang="ts">
withDefaults(
  defineProps<{ tone?: "neutral" | "ok" | "warn" | "err" | "accent" }>(),
  { tone: "neutral" },
);
</script>

<template>
  <span
    class="inline-flex items-center gap-1 rounded px-1.5 text-[11px] leading-[18px]"
    :class="
      {
        neutral: 'bg-raised text-muted',
        ok: 'bg-ok/15 text-ok-fg',
        warn: 'bg-warn/15 text-warn-fg',
        err: 'bg-err/15 text-err-fg',
        accent: 'bg-accent-soft text-accent-text',
      }[tone]
    "
  >
    <slot />
  </span>
</template>
