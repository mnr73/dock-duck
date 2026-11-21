<script setup lang="ts">
import { Toaster } from "vue-sonner";

import { useTheme } from "@/composables/useTheme";

const theme = useTheme();
</script>

<template>
  <Toaster
    position="bottom-right"
    :theme="theme === 'auto' ? 'system' : theme"
    :toast-options="{
      style: {
        background: 'var(--dd-surface)',
        color: 'var(--dd-fg)',
        border: '1px solid var(--dd-line-strong)',
        fontSize: '12.5px',
      },
    }"
  />
</template>
