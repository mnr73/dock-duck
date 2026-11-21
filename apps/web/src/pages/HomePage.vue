<script setup lang="ts">
import { IconDocker } from "@/icons";

import { type Health, useApi } from "@/api/client";

const { data: health, error, isFetching } = useApi("/health").json<Health>();
</script>

<template>
  <main class="flex h-full flex-col items-center justify-center gap-3">
    <IconDocker class="size-10 text-(--dd-accent)" />
    <h1 class="text-2xl font-semibold">DockDuck</h1>
    <p class="text-sm text-(--dd-muted)">Your local stacks, under control.</p>
    <p class="font-mono text-xs text-(--dd-muted)">
      <template v-if="isFetching">connecting…</template>
      <template v-else-if="error">server unreachable</template>
      <template v-else>server {{ health?.version }} · ok</template>
    </p>
  </main>
</template>
