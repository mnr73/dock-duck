<script setup lang="ts">
import { ref, watch } from "vue";

import { IconClose } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Badge from "@/components/ui/Badge.vue";
import IconBtn from "@/components/ui/IconBtn.vue";
import Spinner from "@/components/ui/Spinner.vue";

import { api, errorMessage } from "@/api/client";
import type { ContainerDetail } from "@/api/types";
import { useSystemStore } from "@/stores/system";

const props = defineProps<{ projectId: string; service: string | null }>();
const emit = defineEmits<{ close: [] }>();
const system = useSystemStore();

const details = ref<ContainerDetail[]>([]);
const loading = ref(false);
const error = ref("");

watch(
  () => props.service,
  async (svc) => {
    if (!svc) return;
    loading.value = true;
    error.value = "";
    details.value = [];
    try {
      details.value =
        (await api.get<ContainerDetail[]>(
          `/projects/${props.projectId}/services/${svc}`,
        )) ?? [];
    } catch (e) {
      error.value = errorMessage(e);
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);

const when = (s?: string) => (s ? new Date(s).toLocaleString() : "—");
</script>

<template>
  <Transition name="dd-fade">
    <aside
      v-if="service"
      aria-label="Inspect"
      class="absolute inset-y-0 right-0 z-30 flex w-[440px] max-w-full flex-col border-l border-line-strong bg-surface shadow-pop"
    >
      <header class="flex items-center gap-2 border-b border-line px-4 py-2.5">
        <h2 class="flex-1 text-[14px] font-semibold">
          Inspect · <span class="font-mono">{{ service }}</span>
        </h2>
        <IconBtn label="Close inspector" size="sm" @click="emit('close')"
          ><IconClose class="size-4"
        /></IconBtn>
      </header>
      <div class="min-h-0 flex-1 overflow-auto p-4">
        <div v-if="loading" class="flex items-center gap-2 text-muted">
          <Spinner class="size-4" /> Loading…
        </div>
        <Alert v-else-if="error" tone="err">{{ error }}</Alert>
        <p v-else-if="!details.length" class="text-muted">
          This service has no containers yet.
        </p>

        <section v-for="c in details" :key="c.id" class="mb-6 last:mb-0">
          <h3
            class="mb-2 flex items-center gap-2 font-mono text-[12.5px] font-semibold"
          >
            {{ c.name }}
            <Badge
              :tone="
                c.state === 'running' ? 'ok' : c.exit_code ? 'err' : 'neutral'
              "
              >{{ c.state }}</Badge
            >
            <Badge
              v-if="c.health"
              :tone="c.health === 'healthy' ? 'ok' : 'warn'"
              >{{ c.health }}</Badge
            >
          </h3>
          <dl class="grid grid-cols-[110px_1fr] gap-x-3 gap-y-1.5 text-[12px]">
            <dt class="text-muted">image</dt>
            <dd class="font-mono break-all">{{ c.image }}</dd>
            <dt class="text-muted">id</dt>
            <dd class="font-mono">{{ c.id }}</dd>
            <dt class="text-muted">command</dt>
            <dd class="font-mono break-all">{{ c.command || "—" }}</dd>
            <dt v-if="c.working_dir" class="text-muted">workdir</dt>
            <dd v-if="c.working_dir" class="font-mono">{{ c.working_dir }}</dd>
            <dt class="text-muted">user</dt>
            <dd class="font-mono">{{ c.user || "image default" }}</dd>
            <dt class="text-muted">created</dt>
            <dd>{{ when(c.created) }}</dd>
            <dt class="text-muted">started</dt>
            <dd>{{ when(c.started_at) }}</dd>
            <dt v-if="c.finished_at" class="text-muted">finished</dt>
            <dd v-if="c.finished_at">{{ when(c.finished_at) }}</dd>
            <dt class="text-muted">exit code</dt>
            <dd class="font-mono" :class="c.exit_code ? 'text-err-fg' : ''">
              {{ c.exit_code }}<span v-if="c.oom_killed"> · OOM killed</span>
            </dd>
            <dt class="text-muted">restarts</dt>
            <dd>
              {{ c.restart_count }} · policy
              <span class="font-mono">{{ c.restart_policy }}</span>
            </dd>
            <dt v-if="c.privileged" class="text-muted">privileged</dt>
            <dd v-if="c.privileged" class="text-warn-fg">yes</dd>
          </dl>
          <p
            v-if="c.error"
            class="mt-2 rounded-md border border-danger-line bg-danger-bg p-2 font-mono text-[11.5px] text-err-fg"
          >
            {{ c.error }}
          </p>
          <p
            v-if="c.health_output"
            class="mt-2 rounded-md bg-raised p-2 font-mono text-[11.5px] text-muted"
          >
            last health check: {{ c.health_output }}
          </p>

          <h4 class="mt-3 mb-1 text-[11.5px] font-medium text-muted">Ports</h4>
          <p v-if="!c.ports.length" class="text-[12px] text-dim">
            None published.
          </p>
          <p v-for="(p, i) in c.ports" :key="i" class="font-mono text-[12px]">
            {{ p.host_ip || "0.0.0.0" }}:{{ p.host_port }} →
            {{ p.container_port }}/{{ p.proto }}
          </p>

          <h4 class="mt-3 mb-1 text-[11.5px] font-medium text-muted">Mounts</h4>
          <p v-if="!c.mounts.length" class="text-[12px] text-dim">None.</p>
          <div
            v-for="(m, i) in c.mounts"
            :key="i"
            class="mb-1 font-mono text-[11.5px]"
          >
            <span class="break-all">{{
              m.type === "volume" ? m.name : system.short(m.source)
            }}</span>
            <span class="text-dim"> → </span
            ><span class="break-all">{{ m.destination }}</span>
            <span class="ml-1 text-dim"
              >{{ m.type }}{{ m.rw ? "" : " · ro" }}</span
            >
          </div>

          <h4 class="mt-3 mb-1 text-[11.5px] font-medium text-muted">
            Networks
          </h4>
          <p
            v-for="n in c.networks"
            :key="n.name"
            class="font-mono text-[12px]"
          >
            {{ n.name }} <span class="text-muted">{{ n.ip }}</span>
            <span v-if="n.aliases?.length" class="text-dim">
              · {{ n.aliases.join(", ") }}</span
            >
          </p>
        </section>
        <p class="mt-4 text-[11.5px] text-dim">
          Environment variables are not shown here; they are handled in the
          Environment tab.
        </p>
      </div>
    </aside>
  </Transition>
</template>
