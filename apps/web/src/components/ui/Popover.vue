<script setup lang="ts">
import { useTemplateRef } from "vue";

import { onClickOutside, onKeyStroke } from "@vueuse/core";

const open = defineModel<boolean>("open", { default: false });
withDefaults(defineProps<{ align?: "start" | "end"; panelClass?: string }>(), {
  align: "start",
});

const root = useTemplateRef<HTMLElement>("root");
onClickOutside(root, () => (open.value = false));
onKeyStroke("Escape", () => (open.value = false));
</script>

<template>
  <div ref="root" class="relative">
    <slot name="trigger" :open="open" :toggle="() => (open = !open)" />
    <Transition name="dd-fade">
      <div
        v-if="open"
        class="absolute z-40 mt-1 rounded-lg border border-line-strong bg-surface shadow-pop"
        :class="[align === 'end' ? 'right-0' : 'left-0', panelClass]"
      >
        <slot :close="() => (open = false)" />
      </div>
    </Transition>
  </div>
</template>
