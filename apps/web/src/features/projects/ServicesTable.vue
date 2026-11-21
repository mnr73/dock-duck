<script setup lang="ts">
import { computed } from "vue";

import { IconExternal } from "@/icons";

import Badge from "@/components/ui/Badge.vue";
import Spinner from "@/components/ui/Spinner.vue";

import type { ServiceInfo } from "@/api/types";
import { formatBytes, formatCpu, portUrl } from "@/utils/format";
import { serviceTone } from "@/utils/status";

import ServiceRowMenu from "./ServiceRowMenu.vue";
import StatusLabel from "./StatusLabel.vue";

const props = defineProps<{
  services: ServiceInfo[];
  usage: Record<string, { cpu: number; mem: number } | undefined>;
  /** services with an action in flight */
  busy: string[];
  highlight?: string;
}>();
const emit = defineEmits<{
  action: [service: string, action: "start" | "stop" | "restart"];
  inspect: [service: string];
  logs: [service: string];
  exec: [service: string];
}>();

const rows = computed(() =>
  props.services.map((s) => ({
    s,
    ports: s.ports.filter((p) => p.host_port),
    running: s.containers.filter((c) => c.state === "running").length,
    health: s.health,
  })),
);
</script>

<template>
  <table class="w-full border-collapse text-left">
    <thead>
      <tr class="border-b border-line text-[11.5px] text-muted">
        <th class="w-8 py-2 pl-4 font-normal"></th>
        <th class="py-2 pr-4 font-normal">service</th>
        <th class="py-2 pr-4 font-normal">state</th>
        <th class="py-2 pr-4 font-normal">image</th>
        <th class="py-2 pr-4 font-normal">ports (host→container)</th>
        <th class="py-2 pr-4 text-right font-normal">cpu</th>
        <th class="py-2 pr-4 text-right font-normal">memory</th>
        <th class="w-10 py-2 pr-3"></th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="r in rows"
        :key="r.s.name"
        class="border-b border-line/60 hover:bg-hover/40"
        :class="highlight === r.s.name && 'bg-accent-soft/60'"
      >
        <td class="py-2 pl-4">
          <Spinner v-if="busy.includes(r.s.name)" class="size-3.5 text-muted" />
        </td>
        <td class="py-2 pr-4 font-mono font-semibold">
          {{ r.s.name }}
          <span v-if="r.s.containers.length > 1" class="font-normal text-dim"
            >×{{ r.s.containers.length }}</span
          >
        </td>
        <td class="py-2 pr-4 font-mono">
          <span class="inline-flex flex-wrap items-center gap-x-2">
            <StatusLabel :tone="serviceTone(r.s.status)">{{
              r.s.status
            }}</StatusLabel>
            <Badge
              v-if="
                r.health && r.health !== 'healthy' && r.s.status !== 'unhealthy'
              "
              tone="warn"
              >{{ r.health }}</Badge
            >
            <span
              v-else-if="r.health === 'healthy'"
              class="text-[11px] text-dim"
              >healthy</span
            >
            <span v-if="r.s.containers.length > 1" class="text-[11px] text-dim"
              >{{ r.running }}/{{ r.s.containers.length }}</span
            >
          </span>
        </td>
        <td
          class="max-w-[220px] truncate py-2 pr-4 font-mono text-[12px] text-muted"
          :title="r.s.image"
        >
          {{ r.s.image }}
        </td>
        <td class="py-2 pr-4 font-mono text-[12px]">
          <span v-if="!r.ports.length" class="text-dim">—</span>
          <span
            v-for="(p, i) in r.ports"
            :key="i"
            class="mr-2 inline-flex items-center"
          >
            <a
              :href="portUrl(p.host_port!, p.container_port)"
              target="_blank"
              rel="noopener"
              class="inline-flex items-center gap-1 text-muted hover:text-accent-text"
              :title="`Open ${portUrl(p.host_port!, p.container_port)}`"
            >
              {{ p.host_port }} <span class="text-dim">→</span>
              {{ p.container_port }}
              <IconExternal class="size-3 opacity-60" />
            </a>
          </span>
        </td>
        <td class="py-2 pr-4 text-right font-mono text-[12px] text-muted">
          {{
            props.usage[r.s.name] ? formatCpu(props.usage[r.s.name]!.cpu) : "—"
          }}
        </td>
        <td class="py-2 pr-4 text-right font-mono text-[12px] text-muted">
          {{
            props.usage[r.s.name]
              ? formatBytes(props.usage[r.s.name]!.mem)
              : "—"
          }}
        </td>
        <td class="py-2 pr-3 text-right">
          <ServiceRowMenu
            :service="r.s"
            :busy="busy.includes(r.s.name)"
            @action="(a) => emit('action', r.s.name, a)"
            @inspect="emit('inspect', r.s.name)"
            @logs="emit('logs', r.s.name)"
            @exec="emit('exec', r.s.name)"
          />
        </td>
      </tr>
    </tbody>
  </table>
</template>
