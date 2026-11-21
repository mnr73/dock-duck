<script setup lang="ts">
import { type Ref, computed, nextTick, watch } from "vue";

import { useEventListener, useVirtualList } from "@vueuse/core";

import { IconFollow } from "@/icons";

import {
  type LogLine,
  type Matcher,
  formatTime,
  highlight,
  serviceColor,
} from "@/utils/logs";

const props = defineProps<{
  lines: LogLine[];
  matcher: Matcher;
  showTime: boolean;
  wrap: boolean;
  follow: boolean;
  /** every service name, for stable colours */
  services: string[];
  showService: boolean;
}>();
const emit = defineEmits<{ "update:follow": [v: boolean] }>();
const selected = defineModel<Set<number>>("selected", { required: true });

const ROW = 20;
const list = computed(() => props.lines);
const {
  list: visible,
  containerProps,
  wrapperProps,
  scrollTo,
} = useVirtualList(list, { itemHeight: ROW, overscan: 30 });

const container = containerProps.ref as Ref<HTMLElement | null>;
const svcWidth = computed(() =>
  Math.min(
    24,
    props.lines.reduce((n, l) => Math.max(n, l.s.length), 0),
  ),
);

// Follow: keep the view pinned to the newest line; scrolling up by hand turns it off,
// scrolling back to the bottom turns it on again.
function atBottom(el: HTMLElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight < ROW * 1.5;
}
function onScroll() {
  const el = container.value;
  if (!el) return;
  const bottom = atBottom(el);
  if (bottom !== props.follow && !programmatic) emit("update:follow", bottom);
}
let programmatic = false;
async function toBottom() {
  await nextTick();
  const el = container.value;
  if (!el) return;
  programmatic = true;
  el.scrollTop = el.scrollHeight;
  requestAnimationFrame(() => (programmatic = false));
}
watch(
  () => props.lines.length,
  () => {
    if (props.follow) toBottom();
  },
);
watch(
  () => props.follow,
  (f) => {
    if (f) toBottom();
  },
);
defineExpose({ toBottom, scrollTo });

// When the list shrinks (filter, search, clear) the old scroll offset can point past its
// end, so the virtual window renders nothing until a scroll event happens. Clamp the
// offset and let VueUse recompute the window right away.
watch(
  () => props.lines,
  async () => {
    await nextTick();
    const el = container.value;
    if (!el) return;
    const max = Math.max(0, props.lines.length * ROW - el.clientHeight);
    if (el.scrollTop > max) {
      programmatic = true;
      el.scrollTop = max;
      requestAnimationFrame(() => (programmatic = false));
    }
    containerProps.onScroll();
  },
);

// Selection: click toggles one line, shift-click selects a range.
let anchor = -1;
function pick(l: LogLine, e: MouseEvent) {
  const next = new Set(selected.value);
  if (e.shiftKey && anchor >= 0) {
    const a = props.lines.findIndex((x) => x.id === anchor);
    const b = props.lines.findIndex((x) => x.id === l.id);
    if (a >= 0 && b >= 0) {
      for (let i = Math.min(a, b); i <= Math.max(a, b); i++)
        next.add(props.lines[i].id);
    }
  } else if (next.has(l.id)) {
    next.delete(l.id);
  } else {
    next.add(l.id);
  }
  anchor = l.id;
  selected.value = next;
}

// drop selections of lines that no longer exist (buffer trimmed / cleared)
watch(
  () => props.lines[0]?.id,
  (first) => {
    if (first === undefined) {
      if (selected.value.size) selected.value = new Set();
      return;
    }
    if ([...selected.value].some((id) => id < first))
      selected.value = new Set([...selected.value].filter((id) => id >= first));
  },
);

useEventListener(container, "wheel", (e: WheelEvent) => {
  if (e.deltaY < 0 && props.follow) emit("update:follow", false);
});

const LEVEL_TEXT: Record<string, string> = {
  error: "text-[#ff8f8f]",
  warn: "text-[#f2c672]",
  debug: "text-[#8a8a8a]",
  info: "",
};
</script>

<template>
  <div
    class="relative min-h-0 flex-1 bg-term font-mono text-[12px] text-term-fg"
  >
    <div
      ref="container"
      :style="containerProps.style"
      class="absolute inset-0 overflow-auto"
      :class="wrap ? '' : 'overflow-x-auto'"
      @scroll="
        containerProps.onScroll();
        onScroll();
      "
    >
      <div :style="wrapperProps.style" :class="wrap ? '' : 'w-max min-w-full'">
        <div
          v-for="{ data: l, index } in visible"
          :key="l.id"
          class="group flex cursor-default items-start gap-2 px-3 leading-[20px] hover:bg-white/5"
          :class="[
            selected.has(l.id) && 'bg-accent/15 hover:bg-accent/20',
            wrap ? 'min-h-[20px]' : 'h-[20px]',
          ]"
          :style="wrap ? undefined : { height: ROW + 'px' }"
          :data-index="index"
          @click="pick(l, $event)"
        >
          <span v-if="showTime" class="shrink-0 text-[#6e6e6e] select-none">{{
            formatTime(l.t) || "            "
          }}</span>
          <span
            v-if="showService"
            class="shrink-0 select-none"
            :style="{
              color: serviceColor(l.s, services),
              width: svcWidth + 'ch',
            }"
            :title="l.s"
          >
            {{ l.s }}
          </span>
          <span
            class="min-w-0 flex-1"
            :class="[
              wrap ? 'break-all whitespace-pre-wrap' : 'whitespace-pre',
              LEVEL_TEXT[l.level],
            ]"
          >
            <template v-for="(p, i) in highlight(l.m, matcher)" :key="i">
              <mark
                v-if="p.hit"
                class="rounded-[2px] bg-[#f2b94b] px-px text-[#0a0a0a]"
                >{{ p.text }}</mark
              >
              <template v-else>{{ p.text }}</template>
            </template>
          </span>
        </div>
      </div>
    </div>

    <button
      v-if="!follow && lines.length"
      type="button"
      class="absolute right-4 bottom-3 flex items-center gap-1.5 rounded-md border border-term-line bg-[#141414] px-2.5 py-1 text-[11.5px] text-term-fg shadow-pop hover:bg-[#1a1a1a]"
      @click="emit('update:follow', true)"
    >
      <IconFollow class="size-3.5" /> Jump to latest
    </button>
    <slot />
  </div>
</template>
