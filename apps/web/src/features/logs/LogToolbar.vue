<script setup lang="ts">
import { computed, ref } from "vue";

import { onClickOutside } from "@vueuse/core";

import {
  IconChevronDown,
  IconClear,
  IconDownload,
  IconFollow,
  IconPause,
  IconPlay,
  IconRegex,
  IconSearch,
  IconTimestamp,
  IconWrap,
} from "@/icons";

import Btn from "@/components/ui/Btn.vue";
import IconBtn from "@/components/ui/IconBtn.vue";
import Popover from "@/components/ui/Popover.vue";
import StatusDot from "@/components/ui/StatusDot.vue";

import type { ServiceInfo } from "@/api/types";
import type { LevelFilter } from "@/utils/logs";
import { serviceColor } from "@/utils/logs";
import { serviceTone } from "@/utils/status";

const props = defineProps<{
  services: ServiceInfo[];
  live: boolean;
  ended: boolean;
  /** the project has no containers: nothing to stream */
  idle?: boolean;
  visibleCount: number;
  totalCount: number;
  searchError: string;
}>();
defineEmits<{ pause: []; resume: []; clear: []; download: []; restart: [] }>();

const selectedServices = defineModel<string[]>("selectedServices", {
  required: true,
});
const query = defineModel<string>("query", { required: true });
const regex = defineModel<boolean>("regex", { required: true });
const level = defineModel<LevelFilter>("level", { required: true });
const since = defineModel<string>("since", { required: true });
const showTime = defineModel<boolean>("showTime", { required: true });
const wrap = defineModel<boolean>("wrap", { required: true });
const follow = defineModel<boolean>("follow", { required: true });
const paused = defineModel<boolean>("paused", { required: true });

const svcOpen = ref(false);
const allNames = computed(() => props.services.map((s) => s.name));
const label = computed(() => {
  const n = selectedServices.value.length;
  if (!n) return "All services";
  if (n === 1) return selectedServices.value[0];
  return `${n} services`;
});

function toggle(name: string) {
  const cur = new Set(selectedServices.value);
  if (cur.has(name)) cur.delete(name);
  else cur.add(name);
  selectedServices.value = [...cur];
}

const SINCE = [
  { v: "", l: "Since: start" },
  { v: "5m", l: "Since: 5m" },
  { v: "15m", l: "Since: 15m" },
  { v: "1h", l: "Since: 1h" },
  { v: "6h", l: "Since: 6h" },
  { v: "24h", l: "Since: 24h" },
];

const searchBox = ref<HTMLElement | null>(null);
onClickOutside(searchBox, () => {});
defineExpose({
  focusSearch: () =>
    (
      searchBox.value?.querySelector("input") as HTMLInputElement | null
    )?.focus(),
});
</script>

<template>
  <div
    class="flex shrink-0 flex-wrap items-center gap-x-2 gap-y-1.5 border-b border-line bg-app px-3 py-2"
  >
    <span
      class="flex h-6 items-center gap-1.5 rounded-full border border-line-strong bg-surface px-2.5 text-[11.5px]"
      :title="
        ended ? 'The log process ended' : live ? 'Streaming' : 'Connecting…'
      "
    >
      <StatusDot
        :tone="idle || ended ? 'off' : live && !paused ? 'ok' : 'warn'"
        size="xs"
        :pulse="live && !paused && !ended"
      />
      {{
        idle
          ? "not running"
          : ended
            ? "ended"
            : paused
              ? "paused"
              : live
                ? "live"
                : "connecting"
      }}
    </span>

    <Popover v-model:open="svcOpen" panel-class="w-60 p-1">
      <template #trigger="{ toggle: t }">
        <Btn size="sm" @click="t">
          {{ label }}
          <template #icon
            ><IconChevronDown class="size-3.5 order-last"
          /></template>
        </Btn>
      </template>
      <button
        type="button"
        class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12.5px] hover:bg-hover"
        @click="selectedServices = []"
      >
        <input
          type="checkbox"
          :checked="!selectedServices.length"
          tabindex="-1"
          class="pointer-events-none"
        />
        All services
      </button>
      <button
        v-for="s in services"
        :key="s.name"
        type="button"
        class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12.5px] hover:bg-hover"
        @click="toggle(s.name)"
      >
        <input
          type="checkbox"
          :checked="selectedServices.includes(s.name)"
          tabindex="-1"
          class="pointer-events-none"
        />
        <span
          class="size-2 shrink-0 rounded-full"
          :style="{ background: serviceColor(s.name, allNames) }"
        />
        <span class="min-w-0 flex-1 truncate font-mono">{{ s.name }}</span>
        <StatusDot :tone="serviceTone(s.status)" size="xs" />
      </button>
    </Popover>

    <label
      class="flex h-[26px] items-center gap-1.5 rounded-md border border-line-strong bg-surface px-1.5 text-[12px]"
    >
      <select
        v-model="level"
        aria-label="Level"
        class="bg-transparent outline-none"
      >
        <option value="all">Level: all</option>
        <option value="warn">Level: warn+</option>
        <option value="error">Level: error</option>
      </select>
    </label>
    <label
      class="flex h-[26px] items-center gap-1.5 rounded-md border border-line-strong bg-surface px-1.5 text-[12px]"
    >
      <select
        v-model="since"
        aria-label="Since"
        class="bg-transparent outline-none"
      >
        <option v-for="o in SINCE" :key="o.v" :value="o.v">{{ o.l }}</option>
      </select>
    </label>

    <div
      ref="searchBox"
      class="flex h-[26px] min-w-[200px] flex-1 items-center gap-1.5 rounded-md border px-2 sm:max-w-xs"
      :class="searchError ? 'border-err' : 'border-line-strong bg-surface'"
    >
      <IconSearch class="size-3.5 text-muted" />
      <input
        v-model="query"
        aria-label="Search logs"
        placeholder="Search…  (Ctrl+F)"
        spellcheck="false"
        class="min-w-0 flex-1 bg-transparent font-mono text-[12px] outline-none placeholder:text-dim"
      />
      <span v-if="query" class="shrink-0 text-[11px] text-muted"
        >{{ visibleCount }} {{ visibleCount === 1 ? "match" : "matches" }}</span
      >
      <IconBtn
        label="Regular expression"
        size="sm"
        :active="regex"
        @click="regex = !regex"
        ><IconRegex class="size-3.5"
      /></IconBtn>
    </div>
    <span v-if="searchError" class="text-[11.5px] text-err-fg">{{
      searchError
    }}</span>

    <span class="ml-auto flex items-center gap-1">
      <IconBtn
        label="Timestamps"
        size="sm"
        :active="showTime"
        @click="showTime = !showTime"
        ><IconTimestamp class="size-4"
      /></IconBtn>
      <IconBtn label="Wrap lines" size="sm" :active="wrap" @click="wrap = !wrap"
        ><IconWrap class="size-4"
      /></IconBtn>
      <IconBtn
        label="Follow (auto-scroll)"
        size="sm"
        :active="follow"
        @click="follow = !follow"
        ><IconFollow class="size-4"
      /></IconBtn>
      <span class="mx-0.5 h-4 w-px bg-line-strong" />
      <Btn v-if="ended && !idle" size="sm" @click="$emit('restart')">
        <template #icon><IconPlay class="size-3" /></template>
        Reconnect
      </Btn>
      <Btn
        v-else
        size="sm"
        :disabled="idle"
        @click="paused ? $emit('resume') : $emit('pause')"
      >
        <template #icon
          ><component :is="paused ? IconPlay : IconPause" class="size-3.5"
        /></template>
        {{ paused ? "Resume" : "Pause" }}
      </Btn>
      <IconBtn label="Clear" size="sm" @click="$emit('clear')"
        ><IconClear class="size-4"
      /></IconBtn>
      <IconBtn label="Download logs" size="sm" @click="$emit('download')"
        ><IconDownload class="size-4"
      /></IconBtn>
    </span>
  </div>
</template>
