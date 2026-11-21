<script setup lang="ts">
import StatusLabel from "@/features/projects/StatusLabel.vue";

import type { ServiceInfo } from "@/api/types";
import { serviceTone } from "@/utils/status";

defineProps<{ services: ServiceInfo[] }>();
</script>

<template>
  <table class="w-full border-collapse text-left">
    <thead>
      <tr class="border-b border-line text-[11.5px] text-muted">
        <th class="w-8 py-2 pl-4 font-normal"></th>
        <th class="py-2 font-normal">service</th>
        <th class="py-2 font-normal">state</th>
        <th class="py-2 font-normal">image</th>
        <th class="py-2 pr-4 font-normal">ports</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="s in services"
        :key="s.name"
        class="border-b border-line/60 hover:bg-hover/40"
      >
        <td class="py-2 pl-4"><slot name="lead" :service="s" /></td>
        <td class="py-2 pr-4 font-mono font-semibold">
          {{ s.name }}
          <span v-if="s.containers.length > 1" class="font-normal text-dim"
            >×{{ s.containers.length }}</span
          >
        </td>
        <td class="py-2 pr-4 font-mono">
          <StatusLabel :tone="serviceTone(s.status)">{{
            s.status
          }}</StatusLabel>
        </td>
        <td class="py-2 pr-4 font-mono text-[12px] text-muted">
          {{ s.image }}
        </td>
        <td class="py-2 pr-4 font-mono text-[12px] text-muted">
          <template
            v-for="(p, i) in s.ports.filter((x) => x.host_port)"
            :key="i"
          >
            <span v-if="i">, </span>{{ p.host_port }}→{{ p.container_port }}
          </template>
        </td>
      </tr>
    </tbody>
  </table>
</template>
