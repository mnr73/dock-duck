<script setup lang="ts">
import { computed } from "vue";

import { IconPlay } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";

import type { ProjectSummary } from "@/api/types";
import { useJobsStore } from "@/stores/jobs";

const props = defineProps<{ project: ProjectSummary; what: string }>();
const open = defineModel<boolean>({ required: true });

const jobs = useJobsStore();
const busy = computed(() => !!jobs.running[props.project.id]);

async function apply() {
  // `up -d` recreates only the services whose configuration changed
  const started = await jobs.run({
    projectId: props.project.id,
    projectName: props.project.name,
    action: "up",
  });
  if (started) open.value = false;
}
</script>

<template>
  <div v-if="open && project.running > 0" class="px-4 pt-3">
    <Alert tone="info">
      <div class="flex items-center gap-3">
        <span class="flex-1">
          {{ what }} saved. Running containers keep the old configuration until
          they are recreated.
        </span>
        <Btn size="sm" variant="primary" :disabled="busy" @click="apply">
          <IconPlay class="size-3.5" /> Apply (up -d)
        </Btn>
        <Btn size="sm" variant="ghost" @click="open = false">Later</Btn>
      </div>
    </Alert>
  </div>
</template>
