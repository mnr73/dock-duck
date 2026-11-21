<script setup lang="ts">
import type { Component } from "vue";

import Popover from "./Popover.vue";

export interface MenuItem {
  label: string;
  icon?: Component;
  danger?: boolean;
  disabled?: boolean;
  /** draw a divider above this item */
  separator?: boolean;
  /** tooltip, e.g. why the item is disabled */
  hint?: string;
  onSelect: () => void;
}

const open = defineModel<boolean>("open", { default: false });
defineProps<{ items: MenuItem[]; align?: "start" | "end" }>();
</script>

<template>
  <Popover v-model:open="open" :align="align" panel-class="min-w-44 p-1">
    <template #trigger="slot"><slot name="trigger" v-bind="slot" /></template>
    <template #default="{ close }">
      <template v-for="it in items" :key="it.label">
        <div v-if="it.separator" role="separator" class="my-1 h-px bg-line" />
        <button
          type="button"
          role="menuitem"
          :title="it.hint"
          :disabled="it.disabled"
          class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[12.5px] hover:bg-hover disabled:opacity-45"
          :class="it.danger ? 'text-err-fg' : 'text-fg'"
          @click="
            close();
            it.onSelect();
          "
        >
          <component :is="it.icon" v-if="it.icon" class="size-3.5 text-muted" />
          {{ it.label }}
        </button>
      </template>
    </template>
  </Popover>
</template>
