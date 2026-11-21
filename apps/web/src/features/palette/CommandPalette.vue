<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";

import { IconCornerDownLeft, IconSearch } from "@/icons";

import { useCommands } from "@/composables/useCommands";
import { bestScore } from "@/utils/fuzzy";

import type { Command } from "./types";

const open = defineModel<boolean>("open", { required: true });

const commands = useCommands();
const query = ref("");
const active = ref(0);
const input = ref<HTMLInputElement>();
const list = ref<HTMLElement>();

const MAX = 40;

/** Matching commands, best first; with no query the natural order is kept. */
const results = computed<Command[]>(() => {
  const q = query.value.trim();
  if (!q) return commands.value.slice(0, MAX);
  return commands.value
    .map((c) => ({
      c,
      s: bestScore(q, [c.title, ...(c.keywords ?? []), c.hint ?? ""]),
    }))
    .filter((x): x is { c: Command; s: number } => x.s !== null)
    .sort((a, b) => a.s - b.s)
    .slice(0, MAX)
    .map((x) => x.c);
});

/** Group headings only when browsing; a search is one ranked list. */
const grouped = computed(() => !query.value.trim());

/** focus goes back to what had it before the palette opened */
let opener: HTMLElement | null = null;

watch(open, async (v) => {
  if (!v) {
    const back = opener;
    opener = null;
    await nextTick();
    if (back?.isConnected) back.focus();
    return;
  }
  opener =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
  query.value = "";
  active.value = 0;
  await nextTick();
  input.value?.focus();
});
watch(query, () => (active.value = 0));

function run(c: Command | undefined) {
  if (!c) return;
  open.value = false;
  c.run();
}

function move(delta: number) {
  const n = results.value.length;
  if (!n) return;
  active.value = (active.value + delta + n) % n;
  nextTick(() =>
    list.value
      ?.querySelector<HTMLElement>(`[data-index="${active.value}"]`)
      ?.scrollIntoView({ block: "nearest" }),
  );
}

function onKey(e: KeyboardEvent) {
  if (e.key === "ArrowDown") {
    e.preventDefault();
    move(1);
  } else if (e.key === "ArrowUp") {
    e.preventDefault();
    move(-1);
  } else if (e.key === "Enter") {
    e.preventDefault();
    run(results.value[active.value]);
  } else if (e.key === "Escape") {
    e.preventDefault();
    open.value = false;
  } else if (e.key === "Tab") {
    // the input is the only stop: arrows move through the list, so Tab must not leave
    e.preventDefault();
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-50 grid place-items-start justify-center bg-overlay p-4 pt-[14vh]"
      @mousedown.self="open = false"
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        class="flex max-h-[60vh] w-full max-w-[560px] flex-col overflow-hidden rounded-xl border border-line-strong bg-surface shadow-pop"
      >
        <div class="flex items-center gap-2 border-b border-line px-3.5">
          <IconSearch class="size-4 shrink-0 text-dim" />
          <input
            ref="input"
            v-model="query"
            role="combobox"
            aria-expanded="true"
            aria-controls="palette-list"
            :aria-activedescendant="
              results.length ? `palette-${active}` : undefined
            "
            aria-label="Search commands"
            placeholder="Type a command or project name"
            spellcheck="false"
            autocomplete="off"
            class="h-11 min-w-0 flex-1 bg-transparent text-[14px] text-fg outline-none placeholder:text-dim"
            @keydown="onKey"
          />
          <kbd
            class="rounded border border-line-strong px-1.5 text-[11px] text-dim"
            >Esc</kbd
          >
        </div>

        <ul
          id="palette-list"
          ref="list"
          role="listbox"
          class="min-h-0 flex-1 overflow-auto p-1.5"
        >
          <template v-for="(c, i) in results" :key="c.id">
            <li
              v-if="grouped && (i === 0 || results[i - 1]!.group !== c.group)"
              role="presentation"
              class="px-2.5 pt-2 pb-1 text-[11px] tracking-wide text-dim uppercase"
            >
              {{ c.group }}
            </li>
            <li
              :id="`palette-${i}`"
              :data-index="i"
              role="option"
              :aria-selected="i === active"
              class="flex cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-2 text-[13px]"
              :class="i === active ? 'bg-hover text-fg' : 'text-muted'"
              @mousemove="active = i"
              @click="run(c)"
            >
              <component :is="c.icon" class="size-4 shrink-0" />
              <span class="min-w-0 flex-1 truncate">{{ c.title }}</span>
              <span
                v-if="c.hint"
                class="max-w-[45%] truncate font-mono text-[11px] text-dim"
              >
                {{ c.hint }}
              </span>
              <IconCornerDownLeft
                v-if="i === active"
                class="size-3.5 shrink-0 text-dim"
              />
            </li>
          </template>
          <li
            v-if="!results.length"
            class="px-3 py-6 text-center text-[13px] text-dim"
          >
            Nothing matches “{{ query }}”
          </li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>
