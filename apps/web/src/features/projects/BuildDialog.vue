<script setup lang="ts">
import { computed, ref, watch } from "vue";

import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";

const open = defineModel<boolean>("open", { required: true });
const props = defineProps<{
  projectName: string;
  /** services to build; empty = every service that has a build section */
  services: string[];
}>();
const emit = defineEmits<{
  confirm: [opts: { noCache: boolean; pull: boolean }];
}>();

const noCache = ref(false);
const pull = ref(false);

watch(open, (v) => {
  if (v) {
    noCache.value = false;
    pull.value = false;
  }
});

const target = computed(() =>
  props.services.length
    ? props.services.join(", ")
    : "every service that has a build section",
);

function confirm() {
  emit("confirm", { noCache: noCache.value, pull: pull.value });
  open.value = false;
}
</script>

<template>
  <Dialog v-model:open="open" :title="`Build ${projectName}`" width="460px">
    <div class="flex flex-col gap-3.5">
      <p>
        Builds the images of
        <b class="font-mono text-[12.5px]">{{ target }}</b>. Running containers
        keep using their old image until you run <b>Up</b> again.
      </p>
      <label class="flex items-start gap-2.5">
        <input v-model="noCache" type="checkbox" class="mt-0.5" />
        <span>
          Don’t use the build cache
          <span class="mt-0.5 block text-[12px] text-muted"
            >Slower: every layer is rebuilt. Use it when a cached layer is
            stale.</span
          >
        </span>
      </label>
      <label class="flex items-start gap-2.5">
        <input v-model="pull" type="checkbox" class="mt-0.5" />
        <span>
          Pull newer base images first
          <span class="mt-0.5 block text-[12px] text-muted"
            >Checks the registry for a newer version of each
            <code class="font-mono">FROM</code> image.</span
          >
        </span>
      </label>
    </div>
    <template #footer>
      <Btn @click="open = false">Cancel</Btn>
      <Btn variant="primary" data-autofocus @click="confirm">Build</Btn>
    </template>
  </Dialog>
</template>
