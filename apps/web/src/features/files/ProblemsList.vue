<script setup lang="ts">
import { IconError, IconWarning } from "@/icons";

import type { Problem } from "@/api/types";

defineProps<{ problems: Problem[]; stale?: boolean }>();
defineEmits<{ goto: [line: number] }>();
</script>

<template>
  <ul class="divide-y divide-line text-[12px]" :class="stale && 'opacity-60'">
    <li v-for="(p, i) in problems" :key="i">
      <button
        type="button"
        class="flex w-full items-start gap-2 px-3 py-1.5 text-left hover:bg-hover"
        :class="!p.line && 'cursor-default'"
        @click="p.line && $emit('goto', p.line)"
      >
        <component
          :is="p.severity === 'error' ? IconError : IconWarning"
          class="mt-0.5 size-3.5 shrink-0"
          :class="p.severity === 'error' ? 'text-err' : 'text-warn'"
        />
        <span class="min-w-0 flex-1 font-mono break-words">{{
          p.message
        }}</span>
        <span v-if="p.line" class="shrink-0 text-dim">line {{ p.line }}</span>
      </button>
    </li>
  </ul>
</template>
