<script setup lang="ts">
import StatusDot from "@/components/ui/StatusDot.vue";

import { type Tone, toneText } from "@/utils/status";

defineProps<{ tone: Tone }>();
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 text-[12px]"
    :class="toneText[tone]"
  >
    <StatusDot :tone="tone" size="xs" />
    <slot />
  </span>
</template>
