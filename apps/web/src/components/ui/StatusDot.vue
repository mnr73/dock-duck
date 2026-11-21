<script setup lang="ts">
import type { Tone } from "@/utils/status";

withDefaults(
  defineProps<{ tone: Tone; size?: "xs" | "sm" | "md"; pulse?: boolean }>(),
  { size: "sm" },
);
</script>

<template>
  <span
    class="inline-block shrink-0 rounded-full border"
    :class="[
      { xs: 'size-1.5', sm: 'size-[7px]', md: 'size-[9px]' }[size],
      {
        ok: 'border-ok bg-ok',
        warn: 'border-warn bg-warn',
        err: 'border-err bg-err',
        off: 'border-dim bg-transparent',
      }[tone],
      pulse && 'animate-pulse',
    ]"
  />
</template>
