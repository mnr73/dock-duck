<script setup lang="ts">
import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";

import { useContextsStore } from "@/stores/contexts";
import { useSystemStore } from "@/stores/system";

const contexts = useContextsStore();
const system = useSystemStore();
</script>

<template>
  <div
    v-if="system.serverDown || (contexts.loaded && contexts.disconnected)"
    class="border-b border-line p-3"
  >
    <Alert tone="err">
      <template v-if="system.serverDown">
        <b class="font-semibold">Can’t reach the DockDuck server.</b>
        <span class="text-muted"> Is it still running?</span>
      </template>
      <template v-else>
        <b class="font-semibold">Can’t reach the Docker daemon.</b>
        <span class="text-muted">
          Context
          <code class="font-mono">{{ contexts.selected?.name ?? "—" }}</code>
          isn’t responding<template v-if="contexts.selected?.error"
            >: {{ contexts.selected.error }}</template
          >. Projects are still listed; actions will fail until it is back.
        </span>
      </template>
      <template #action>
        <span class="flex gap-1.5">
          <RouterLink to="/contexts"
            ><Btn size="sm">Switch context</Btn></RouterLink
          >
          <Btn
            size="sm"
            :loading="contexts.loading"
            @click="
              system.load();
              contexts.load();
            "
            >Retry</Btn
          >
        </span>
      </template>
    </Alert>
  </div>
</template>
