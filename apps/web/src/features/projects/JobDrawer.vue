<script setup lang="ts">
import { nextTick, useTemplateRef, watch } from "vue";

import { useClipboard } from "@vueuse/core";

import { IconClose, IconCopy } from "@/icons";

import Btn from "@/components/ui/Btn.vue";
import IconBtn from "@/components/ui/IconBtn.vue";
import Spinner from "@/components/ui/Spinner.vue";

import { useJobsStore } from "@/stores/jobs";

const jobs = useJobsStore();
const body = useTemplateRef<HTMLElement>("body");
const { copy, copied } = useClipboard({ copiedDuring: 1500 });

watch(
  () => jobs.lines.length,
  async () => {
    await nextTick();
    if (body.value) body.value.scrollTop = body.value.scrollHeight;
  },
);
</script>

<template>
  <Transition name="dd-fade">
    <section
      v-if="jobs.drawerOpen && jobs.current"
      aria-label="Operation output"
      class="absolute inset-x-0 bottom-0 z-30 flex h-[42%] min-h-48 flex-col border-t border-line-strong bg-term shadow-pop"
    >
      <header
        class="flex items-center gap-2.5 border-b border-term-line px-3 py-1.5 text-[12px] text-term-fg"
      >
        <Spinner v-if="jobs.isRunning" class="size-3.5 text-accent" />
        <span class="font-mono font-semibold"
          >{{ jobs.projectName }} · {{ jobs.current.action }}</span
        >
        <span
          v-if="jobs.current.services?.length"
          class="font-mono text-[11px] opacity-70"
          >{{ jobs.current.services.join(", ") }}</span
        >
        <span
          class="rounded px-1.5 text-[11px]"
          :class="
            {
              running: 'bg-warn/20 text-warn-fg',
              succeeded: 'bg-ok/20 text-ok-fg',
              failed: 'bg-err/20 text-err-fg',
            }[jobs.current.status]
          "
        >
          {{ jobs.current.status }}
        </span>
        <span class="ml-auto flex items-center gap-1">
          <Btn size="sm" variant="ghost" @click="copy(jobs.lines.join('\n'))">
            <template #icon><IconCopy class="size-3.5" /></template>
            {{ copied ? "Copied" : "Copy" }}
          </Btn>
          <IconBtn
            label="Close output"
            size="sm"
            @click="jobs.drawerOpen = false"
            ><IconClose class="size-4"
          /></IconBtn>
        </span>
      </header>
      <div
        ref="body"
        class="min-h-0 flex-1 overflow-auto px-3 py-2 font-mono text-[12px] leading-[1.55] text-term-fg"
      >
        <p v-if="!jobs.lines.length" class="opacity-60">
          {{ jobs.isRunning ? "Waiting for output…" : "No output." }}
        </p>
        <div
          v-for="(l, i) in jobs.lines"
          :key="i"
          class="whitespace-pre-wrap"
          :class="/error|failed/i.test(l) && 'text-err-fg'"
        >
          {{ l }}
        </div>
        <p
          v-if="jobs.current.error && jobs.current.status === 'failed'"
          class="mt-1 text-err-fg"
        >
          {{ jobs.current.error }}
        </p>
      </div>
    </section>
  </Transition>
</template>
