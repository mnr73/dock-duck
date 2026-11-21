<script setup lang="ts">
import { IconError, IconInfo, IconSuccess, IconWarning } from "@/icons";

withDefaults(defineProps<{ tone?: "info" | "ok" | "warn" | "err" }>(), {
  tone: "info",
});
const icons = {
  info: IconInfo,
  ok: IconSuccess,
  warn: IconWarning,
  err: IconError,
};
</script>

<template>
  <div
    role="alert"
    class="flex gap-2.5 rounded-lg border px-3 py-2.5 text-[12.5px]"
    :class="
      {
        info: 'border-line-strong bg-raised text-fg',
        ok: 'border-ok/40 bg-ok/10 text-fg',
        warn: 'border-warn/40 bg-warn/10 text-fg',
        err: 'border-danger-line bg-danger-bg text-err-fg',
      }[tone]
    "
  >
    <component
      :is="icons[tone]"
      class="mt-0.5 size-4 shrink-0"
      :class="
        {
          info: 'text-info',
          ok: 'text-ok',
          warn: 'text-warn',
          err: 'text-err',
        }[tone]
      "
    />
    <div class="min-w-0 flex-1"><slot /></div>
    <slot name="action" />
  </div>
</template>
