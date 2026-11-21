<script setup lang="ts">
import type { Component } from "vue";

import {
  IconContext,
  IconFolder,
  IconPorts,
  IconProjects,
  IconRegistry,
  IconSettings,
} from "@/icons";

import BrandMark from "@/components/ui/BrandMark.vue";

interface RailItem {
  to: string;
  label: string;
  icon: Component;
  /** extra route-name prefixes that keep this item highlighted */
  match?: string[];
}

const items: RailItem[] = [
  { to: "/", label: "Projects", icon: IconProjects, match: ["/projects"] },
  { to: "/contexts", label: "Docker contexts", icon: IconContext },
  { to: "/roots", label: "Project roots", icon: IconFolder },
  { to: "/ports", label: "Ports", icon: IconPorts },
  { to: "/registries", label: "Registries", icon: IconRegistry },
];
</script>

<template>
  <nav
    aria-label="App"
    class="flex w-13 shrink-0 flex-col items-center gap-1.5 border-r border-line bg-rail py-3"
  >
    <BrandMark class="mb-2 size-8" title="DockDuck" />
    <RouterLink
      v-for="it in items"
      :key="it.to"
      v-slot="{ href, navigate, isActive, isExactActive }"
      :to="it.to"
      custom
    >
      <a
        :href="href"
        :aria-label="it.label"
        :title="it.label"
        :aria-current="
          (it.to === '/' ? isExactActive || isActive : isActive)
            ? 'page'
            : undefined
        "
        class="grid size-9 place-items-center rounded-lg transition-colors"
        :class="
          (
            it.to === '/'
              ? $route.path === '/' || $route.path.startsWith('/projects')
              : isActive
          )
            ? 'bg-raised text-fg'
            : 'text-muted hover:bg-hover hover:text-fg'
        "
        @click="navigate"
      >
        <component :is="it.icon" class="size-[18px]" />
      </a>
    </RouterLink>
    <RouterLink v-slot="{ href, navigate, isActive }" to="/settings" custom>
      <a
        :href="href"
        aria-label="Settings"
        title="Settings"
        class="mt-auto grid size-9 place-items-center rounded-lg transition-colors"
        :class="
          isActive
            ? 'bg-raised text-fg'
            : 'text-muted hover:bg-hover hover:text-fg'
        "
        @click="navigate"
      >
        <IconSettings class="size-[18px]" />
      </a>
    </RouterLink>
  </nav>
</template>
