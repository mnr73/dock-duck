<script setup lang="ts">
import { ref } from "vue";

import { toast } from "vue-sonner";

import { IconCheck, IconContext, IconRefresh } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Badge from "@/components/ui/Badge.vue";
import Btn from "@/components/ui/Btn.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import StatusDot from "@/components/ui/StatusDot.vue";

import PageHeader from "@/features/layout/PageHeader.vue";

import { errorMessage } from "@/api/client";
import { useContextsStore } from "@/stores/contexts";

const contexts = useContextsStore();
const switching = ref("");

async function use(name: string) {
  switching.value = name;
  try {
    await contexts.select(name);
    toast.success(`Now using context ${name}`);
  } catch (e) {
    toast.error(`Couldn’t switch to ${name}`, { description: errorMessage(e) });
  } finally {
    switching.value = "";
  }
}
</script>

<template>
  <div class="flex min-h-full flex-col">
    <PageHeader
      title="Docker contexts"
      subtitle="Projects, logs and terminals use the selected context."
    >
      <template #actions>
        <Btn :loading="contexts.loading" @click="contexts.load()">
          <template #icon><IconRefresh class="size-3.5" /></template>
          Test all
        </Btn>
      </template>
    </PageHeader>

    <div class="flex max-w-[980px] flex-col gap-2.5 p-5">
      <Alert v-if="contexts.error" tone="err">
        Couldn’t list Docker contexts: {{ contexts.error }}
      </Alert>

      <EmptyState
        v-else-if="contexts.loaded && !contexts.list.length"
        title="No Docker contexts found"
        text="Is the docker CLI installed and on the PATH of the DockDuck server?"
      >
        <template #icon><IconContext class="size-5" /></template>
      </EmptyState>

      <div
        v-for="c in contexts.list"
        :key="c.name"
        class="flex flex-col rounded-[9px] border"
        :class="
          c.selected
            ? 'border-accent/50 bg-accent-soft'
            : c.available
              ? 'border-line bg-sidebar'
              : 'border-danger-line bg-danger-bg/40'
        "
      >
        <div class="flex flex-wrap items-center gap-x-4 gap-y-2 px-3.5 py-3">
          <StatusDot :tone="c.available ? 'ok' : 'err'" size="md" />
          <div class="flex min-w-[220px] flex-1 flex-col">
            <span class="flex items-center gap-2">
              <span class="font-mono font-semibold">{{ c.name }}</span>
              <Badge v-if="c.current">CLI default</Badge>
            </span>
            <span
              class="truncate font-mono text-[11.5px] text-muted"
              :title="c.endpoint"
              >{{ c.endpoint }}</span
            >
            <span
              v-if="c.description"
              class="truncate text-[11.5px] text-dim"
              >{{ c.description }}</span
            >
          </div>
          <span
            class="w-36 text-[12px]"
            :class="c.available ? 'text-ok-fg' : 'text-err-fg'"
          >
            {{ c.available ? "Available" : "Unavailable" }}
          </span>
          <span class="w-28 font-mono text-[11.5px] text-muted">{{
            c.version ? `Engine ${c.version}` : ""
          }}</span>

          <span
            v-if="c.selected"
            class="flex h-7 items-center gap-1.5 rounded-md bg-accent-soft px-2.5 text-[12px] font-medium text-accent-text"
          >
            <IconCheck class="size-3.5" /> Selected
          </span>
          <Btn v-else :loading="switching === c.name" @click="use(c.name)"
            >Use</Btn
          >
        </div>
        <p
          v-if="c.error"
          class="mx-3.5 mb-3 ml-9 rounded-md border border-danger-line bg-danger-bg px-2.5 py-2 font-mono text-[11.5px] text-err-fg"
        >
          {{ c.error }}
        </p>
      </div>

      <p class="mt-2 text-[12px] text-dim">
        DockDuck only lists and switches contexts. Switching affects DockDuck,
        not the docker CLI in your terminal.
      </p>
    </div>
  </div>
</template>
