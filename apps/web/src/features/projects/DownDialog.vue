<script setup lang="ts">
import { computed, ref, watch } from "vue";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";
import Spinner from "@/components/ui/Spinner.vue";

import { api, errorMessage } from "@/api/client";
import type { DownPreview } from "@/api/types";

const open = defineModel<boolean>("open", { required: true });
const props = defineProps<{ projectId: string; projectName: string }>();
const emit = defineEmits<{
  confirm: [opts: { volumes: boolean; removeOrphans: boolean }];
}>();

const preview = ref<DownPreview | null>(null);
const loading = ref(false);
const error = ref("");
const withVolumes = ref(false);
const orphans = ref(false);
const typed = ref("");

watch(open, async (v) => {
  if (!v) return;
  withVolumes.value = false;
  orphans.value = false;
  typed.value = "";
  preview.value = null;
  error.value = "";
  loading.value = true;
  try {
    preview.value = await api.get<DownPreview>(
      `/projects/${props.projectId}/down-preview`,
    );
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    loading.value = false;
  }
});

const hasVolumes = computed(() => (preview.value?.volumes.length ?? 0) > 0);
/** Deleting volumes is irreversible: require typing the project name. */
const canConfirm = computed(
  () =>
    !loading.value &&
    (!withVolumes.value || typed.value.trim() === props.projectName),
);

function confirm() {
  emit("confirm", { volumes: withVolumes.value, removeOrphans: orphans.value });
  open.value = false;
}
</script>

<template>
  <Dialog
    v-model:open="open"
    :title="`Take ${projectName} down?`"
    width="480px"
  >
    <div v-if="loading" class="flex items-center gap-2 text-muted">
      <Spinner class="size-4" /> Checking what would be removed…
    </div>
    <div v-else class="flex flex-col gap-3.5">
      <Alert v-if="error" tone="warn"
        >Couldn’t preview the removal ({{ error }}).
        <code class="font-mono">compose down</code> still works.</Alert
      >
      <p v-if="preview">
        Stops and removes
        <b
          >{{ preview.containers.length }}
          {{ preview.containers.length === 1 ? "container" : "containers" }}</b
        >
        <template v-if="preview.networks.length">
          and the network
          <code
            v-for="(n, i) in preview.networks"
            :key="n"
            class="font-mono text-[12px]"
            >{{ i ? ", " : "" }}{{ n }}</code
          >
        </template>
        . Images and named volumes are kept.
      </p>
      <ul
        v-if="preview?.containers.length"
        class="max-h-28 overflow-auto rounded-md border border-line bg-sidebar px-2.5 py-1.5 font-mono text-[11.5px] text-muted"
      >
        <li v-for="c in preview.containers" :key="c">{{ c }}</li>
      </ul>

      <label
        class="flex items-start gap-2.5"
        :class="!hasVolumes && 'opacity-60'"
      >
        <input
          v-model="withVolumes"
          type="checkbox"
          class="mt-0.5 accent-[var(--dd-err)]"
          :disabled="!hasVolumes"
        />
        <span>
          Also remove volumes <code class="font-mono text-[12px]">(-v)</code>
          <span v-if="hasVolumes" class="mt-0.5 block text-[12px] text-err-fg">
            Deletes
            <code v-for="(v, i) in preview!.volumes" :key="v" class="font-mono"
              >{{ i ? ", " : "" }}{{ v }}</code
            >
            — data in them is gone for good.
          </span>
          <span v-else class="mt-0.5 block text-[12px] text-muted"
            >This project has no named volumes.</span
          >
        </span>
      </label>

      <label class="flex items-start gap-2.5">
        <input v-model="orphans" type="checkbox" class="mt-0.5" />
        <span>
          Remove orphan containers
          <span class="mt-0.5 block text-[12px] text-muted"
            >Containers from services that are no longer in the Compose
            file.</span
          >
        </span>
      </label>

      <div v-if="withVolumes" class="flex flex-col gap-1.5">
        <label class="text-[12px] text-muted" for="down-confirm">
          Type <code class="font-mono text-fg">{{ projectName }}</code> to
          confirm
        </label>
        <input
          id="down-confirm"
          v-model="typed"
          autocomplete="off"
          spellcheck="false"
          data-autofocus
          class="h-8 rounded-md border border-line-strong bg-surface px-2.5 font-mono outline-none focus:border-err"
          @keydown.enter="canConfirm && confirm()"
        />
      </div>
    </div>
    <template #footer>
      <Btn @click="open = false">Cancel</Btn>
      <Btn variant="danger" :disabled="!canConfirm" @click="confirm">
        {{ withVolumes ? "Down and delete volumes" : "Down" }}
      </Btn>
    </template>
  </Dialog>
</template>
