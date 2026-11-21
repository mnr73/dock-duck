<script setup lang="ts">
import { useRoute } from "vue-router";

import { IconInfo } from "@/icons";

import EmptyState from "@/components/ui/EmptyState.vue";

defineProps<{ title: string; phase: string }>();
useRoute();
</script>

<template>
  <EmptyState
    :title="title"
    :text="`This part lands in ${phase}. See doc/04-phases.md.`"
  >
    <template #icon><IconInfo class="size-5" /></template>
  </EmptyState>
</template>
