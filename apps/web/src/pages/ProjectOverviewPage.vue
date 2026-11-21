<script setup lang="ts">
import Spinner from "@/components/ui/Spinner.vue";

import ServicesTable from "@/features/projects/ServicesTable.vue";

import type { ProjectDetail, ProjectSummary } from "@/api/types";

defineProps<{ project: ProjectSummary; detail?: ProjectDetail }>();
</script>

<template>
  <div v-if="!detail" class="flex items-center gap-2 p-5 text-muted">
    <Spinner class="size-4" /> Loading services…
  </div>
  <ServicesTable v-else :services="detail.services" />
</template>
