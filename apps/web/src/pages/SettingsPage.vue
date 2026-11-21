<script setup lang="ts">
import { IconThemeDark, IconThemeLight, IconThemeSystem } from "@/icons";

import SegmentedControl from "@/components/ui/SegmentedControl.vue";

import AccountSection from "@/features/auth/AccountSection.vue";
import PageHeader from "@/features/layout/PageHeader.vue";

import { type ThemeChoice, useTheme } from "@/composables/useTheme";

const theme = useTheme();
const options: {
  value: ThemeChoice;
  label: string;
  icon: typeof IconThemeLight;
}[] = [
  { value: "auto", label: "System", icon: IconThemeSystem },
  { value: "light", label: "Light", icon: IconThemeLight },
  { value: "dark", label: "Dark", icon: IconThemeDark },
];
</script>

<template>
  <div class="flex min-h-full flex-col">
    <PageHeader title="Settings" />
    <div class="flex max-w-[720px] flex-col gap-5 p-5">
      <section
        class="flex items-center justify-between gap-4 rounded-[9px] border border-line p-3.5"
      >
        <div>
          <h2 class="font-medium">Theme</h2>
          <p class="text-[12px] text-muted">
            System follows your OS. Logs and terminals stay dark in every theme.
          </p>
        </div>
        <SegmentedControl
          v-model="theme"
          :options="options"
          aria-label="Theme"
        />
      </section>
      <AccountSection />
      <p class="text-[12px] text-dim">
        Scan, terminal and default-context preferences arrive with Phase 9.
      </p>
    </div>
  </div>
</template>
