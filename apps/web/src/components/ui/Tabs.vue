<script setup lang="ts">
export interface TabItem {
  key: string;
  label: string;
  to: string;
  /** small accent dot, e.g. unsaved changes */
  dot?: boolean;
}

defineProps<{ items: TabItem[]; ariaLabel?: string }>();
</script>

<template>
  <nav
    :aria-label="ariaLabel ?? 'Sections'"
    class="flex overflow-x-auto border-b border-line px-2 text-[12.5px]"
  >
    <RouterLink
      v-for="t in items"
      :key="t.key"
      v-slot="{ isExactActive: isActive, href, navigate }"
      :to="t.to"
      custom
    >
      <a
        :href="href"
        :aria-current="isActive ? 'page' : undefined"
        class="relative flex items-center gap-1 px-2.5 py-2 whitespace-nowrap hover:text-fg"
        :class="isActive ? 'text-fg' : 'text-muted'"
        @click="navigate"
      >
        {{ t.label }}
        <span v-if="t.dot" class="text-accent">●</span>
        <span
          v-if="isActive"
          class="absolute inset-x-1 bottom-0 h-0.5 rounded bg-accent"
        />
      </a>
    </RouterLink>
  </nav>
</template>
