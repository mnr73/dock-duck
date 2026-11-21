<script setup lang="ts">
import { nextTick, useId, useTemplateRef, watch } from "vue";

import { onKeyStroke } from "@vueuse/core";

import { IconClose } from "@/icons";

import IconBtn from "./IconBtn.vue";

const open = defineModel<boolean>("open", { required: true });
const props = withDefaults(
  defineProps<{ title: string; width?: string; dismissible?: boolean }>(),
  {
    width: "460px",
    dismissible: true,
  },
);
const emit = defineEmits<{ close: [] }>();

const titleId = useId();
const panel = useTemplateRef<HTMLElement>("panel");

function close() {
  if (!props.dismissible) return;
  open.value = false;
  emit("close");
}

onKeyStroke("Escape", (e) => {
  if (open.value) {
    e.preventDefault();
    close();
  }
});

const FOCUSABLE =
  "a[href], button:not([disabled]), input:not([disabled]):not([type=hidden]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex='-1'])";

/** Tab and Shift+Tab wrap inside the dialog: nothing behind a modal is reachable. */
function trap(e: KeyboardEvent) {
  if (e.key !== "Tab" || !panel.value) return;
  const items = [
    ...panel.value.querySelectorAll<HTMLElement>(FOCUSABLE),
  ].filter((el) => el.offsetParent !== null);
  if (!items.length) {
    e.preventDefault();
    panel.value.focus();
    return;
  }
  const first = items[0]!;
  const last = items[items.length - 1]!;
  const at = document.activeElement;
  if (e.shiftKey && (at === first || at === panel.value)) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && at === last) {
    e.preventDefault();
    first.focus();
  } else if (!panel.value.contains(at)) {
    e.preventDefault();
    first.focus();
  }
}

/** what had focus before the dialog opened; it gets it back on close */
let opener: HTMLElement | null = null;

watch(open, async (v) => {
  if (!v) {
    const back = opener;
    opener = null;
    // after the transition, the opener may have been re-rendered
    await nextTick();
    if (back?.isConnected) back.focus();
    return;
  }
  opener =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
  if (v) {
    await nextTick();
    const focusable = panel.value?.querySelector<HTMLElement>(
      "[data-autofocus], input, button:not([aria-label='Close'])",
    );
    (focusable ?? panel.value)?.focus();
  }
});
</script>

<template>
  <Teleport to="body">
    <Transition name="dd-fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 grid place-items-center bg-overlay p-4"
        @mousedown.self="close"
      >
        <div
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          tabindex="-1"
          class="flex max-h-[90vh] w-full flex-col rounded-xl border border-line-strong bg-surface shadow-pop outline-none"
          :style="{ maxWidth: width }"
          @keydown="trap"
        >
          <header
            class="flex items-center gap-2 border-b border-line px-4 py-3"
          >
            <h2 :id="titleId" class="flex-1 text-[14px] font-semibold">
              {{ title }}
            </h2>
            <IconBtn v-if="dismissible" label="Close" size="sm" @click="close"
              ><IconClose class="size-4"
            /></IconBtn>
          </header>
          <div class="min-h-0 flex-1 overflow-auto px-4 py-4"><slot /></div>
          <footer
            v-if="$slots.footer"
            class="flex justify-end gap-2 border-t border-line px-4 py-3"
          >
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style>
.dd-fade-enter-active,
.dd-fade-leave-active {
  transition: opacity 0.12s ease;
}
.dd-fade-enter-from,
.dd-fade-leave-to {
  opacity: 0;
}
</style>
