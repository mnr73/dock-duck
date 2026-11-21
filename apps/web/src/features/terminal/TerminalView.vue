<script setup lang="ts">
import { computed, toRef, useTemplateRef, watch } from "vue";

import "@xterm/xterm/css/xterm.css";

import Btn from "@/components/ui/Btn.vue";

import { useTerminal } from "@/composables/useTerminal";

const props = defineProps<{ id: string; active: boolean; fontSize: number }>();
const emit = defineEmits<{
  state: [state: string, code: number | null];
  title: [title: string];
  close: [];
}>();

const host = useTemplateRef<HTMLElement>("host");
// the terminal surface is dark in every theme, like the log panel
const theme = () => ({
  background: "#050505",
  foreground: "#d4d4d4",
  cursor: "#f2b94b",
  selectionBackground: "#f2b94b44",
});

const t = useTerminal(host, () => props.id, {
  fontSize: toRef(props, "fontSize"),
  fontFamily:
    '"Geist Mono Variable", ui-monospace, "DejaVu Sans Mono", monospace',
  theme,
});

watch([t.state, t.exitCode], ([s, c]) => emit("state", s, c));
watch(t.title, (s) => emit("title", s));
// a hidden tab has no size: refit and focus when it becomes the visible one
watch(
  () => props.active,
  (a) => {
    if (!a) return;
    requestAnimationFrame(() => {
      t.refit();
      t.focus();
    });
  },
);

const banner = computed(() => {
  switch (t.state.value) {
    case "exited": {
      const code = t.exitCode.value;
      const ok = code === 0;
      return {
        text: ok
          ? "Session ended."
          : `Session ended with exit code ${code ?? "?"}.`,
        // a failure (docker error, crashed command) must stand out from a normal exit
        tone: ok ? "bg-raised/40 text-muted" : "bg-danger-bg text-err-fg",
      };
    }
    case "taken":
      return {
        text: "This terminal is open in another tab or window, which now controls it.",
        tone: "bg-warn/10 text-warn-fg",
      };
    case "gone":
      return {
        text: "This session no longer exists (DockDuck was restarted, which ends its shells).",
        tone: "bg-raised/40 text-muted",
      };
    case "lost":
      return {
        text: "Connection to DockDuck lost. The shell is still running; reconnecting…",
        tone: "bg-warn/10 text-warn-fg",
      };
    default:
      return null;
  }
});
const canReconnect = computed(
  () => t.state.value === "lost" || t.state.value === "taken",
);

defineExpose({
  focus: t.focus,
  copy: () => {
    const s = t.selection();
    if (s) void navigator.clipboard?.writeText(s);
    return !!s;
  },
  paste: async () => {
    const s = await navigator.clipboard?.readText();
    if (s) t.paste(s);
  },
  clear: t.clear,
});
</script>

<template>
  <div
    class="relative flex min-h-0 flex-1 flex-col bg-term"
    :class="!active && 'hidden'"
  >
    <div
      ref="host"
      class="min-h-0 flex-1 overflow-hidden px-2 py-1.5"
      @click="t.focus()"
    />
    <div
      v-if="banner"
      class="flex items-center gap-3 border-t border-term-line px-3 py-2 text-[12.5px]"
      :class="banner.tone"
      role="status"
    >
      <span class="flex-1">{{ banner.text }}</span>
      <Btn v-if="canReconnect" size="sm" @click="t.reconnect()">Reconnect</Btn>
      <Btn
        v-if="t.state.value === 'exited' || t.state.value === 'gone'"
        size="sm"
        @click="emit('close')"
        >Close tab</Btn
      >
    </div>
  </div>
</template>
