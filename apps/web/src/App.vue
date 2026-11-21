<script setup lang="ts">
import Toaster from "@/components/ui/Toaster.vue";

import AppShell from "@/features/layout/AppShell.vue";

import { useTheme } from "@/composables/useTheme";

useTheme(); // applies the saved/system theme class on <html>
</script>

<template>
  <AppShell />
  <Toaster />
</template>
