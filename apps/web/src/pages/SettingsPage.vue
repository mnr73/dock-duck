<script setup lang="ts">
import { computed, onMounted } from "vue";

import { RouterLink } from "vue-router";

import { IconThemeDark, IconThemeLight, IconThemeSystem } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import SegmentedControl from "@/components/ui/SegmentedControl.vue";
import Spinner from "@/components/ui/Spinner.vue";
import Switch from "@/components/ui/Switch.vue";

import AccountSection from "@/features/auth/AccountSection.vue";
import PageHeader from "@/features/layout/PageHeader.vue";
import ScanDepthField from "@/features/settings/ScanDepthField.vue";
import SettingsRow from "@/features/settings/SettingsRow.vue";
import SettingsSection from "@/features/settings/SettingsSection.vue";

import { useServerSettings } from "@/composables/useServerSettings";
import { type ThemeChoice, useTheme } from "@/composables/useTheme";
import { useContextsStore } from "@/stores/contexts";
import { ACCENTS, type Density, usePrefsStore } from "@/stores/prefs";
import { useProjectsStore } from "@/stores/projects";
import { type UserChoice, useTerminalsStore } from "@/stores/terminals";

const theme = useTheme();
const prefs = usePrefsStore();
const terminals = useTerminalsStore();
const contexts = useContextsStore();
const projects = useProjectsStore();
const server = useServerSettings();

onMounted(() => {
  server.load();
  if (!contexts.loaded) contexts.load();
});

const themeOptions: {
  value: ThemeChoice;
  label: string;
  icon: typeof IconThemeLight;
}[] = [
  { value: "auto", label: "System", icon: IconThemeSystem },
  { value: "light", label: "Light", icon: IconThemeLight },
  { value: "dark", label: "Dark", icon: IconThemeDark },
];
const densityOptions: { value: Density; label: string }[] = [
  { value: "comfortable", label: "Comfortable" },
  { value: "compact", label: "Compact" },
];
const userOptions: { value: UserChoice; label: string }[] = [
  { value: "default", label: "Container default" },
  { value: "root", label: "root (UID 0)" },
  { value: "uid1000", label: "UID 1000" },
  { value: "custom", label: "Custom" },
];

/** "" = the context used last; a context that Docker no longer lists is kept visible */
const contextOptions = computed(() => {
  const names = contexts.list.map((c) => c.name);
  const saved = server.settings.value?.default_context;
  if (saved && !names.includes(saved)) names.push(saved);
  return names;
});
const savedContextMissing = computed(() => {
  const saved = server.settings.value?.default_context;
  return (
    !!saved && contexts.loaded && !contexts.list.some((c) => c.name === saved)
  );
});

async function saveDepth(depth: number) {
  if (await server.save({ scan_depth: depth })) {
    await projects.load(true);
  }
}
</script>

<template>
  <div class="flex min-h-full flex-col">
    <PageHeader title="Settings" subtitle="Changes save automatically" />
    <div class="flex max-w-[760px] flex-col gap-5 p-5">
      <SettingsSection title="Appearance" hint="Stored in this browser only">
        <SettingsRow
          label="Theme"
          hint="System follows your OS. Logs and terminals stay dark in every theme."
        >
          <SegmentedControl
            v-model="theme"
            :options="themeOptions"
            aria-label="Theme"
          />
        </SettingsRow>
        <SettingsRow
          label="Accent colour"
          hint="Primary actions, focus and live states."
        >
          <div role="radiogroup" aria-label="Accent colour" class="flex gap-2">
            <button
              v-for="a in ACCENTS"
              :key="a.name"
              type="button"
              role="radio"
              :aria-checked="prefs.accent === a.value"
              :aria-label="a.name"
              :title="a.name"
              class="size-5 rounded-full border border-line-strong outline-offset-2"
              :class="prefs.accent === a.value ? 'outline-2 outline-fg' : ''"
              :style="{ background: a.value ?? 'var(--color-accent)' }"
              @click="prefs.accent = a.value"
            />
          </div>
        </SettingsRow>
        <SettingsRow label="Density" hint="Row height in tables.">
          <SegmentedControl
            v-model="prefs.density"
            :options="densityOptions"
            aria-label="Density"
          />
        </SettingsRow>
      </SettingsSection>

      <div v-if="server.loading.value && !server.settings.value" class="py-6">
        <Spinner class="mx-auto size-4 text-muted" />
      </div>
      <Alert v-else-if="server.error.value" tone="err">
        Couldn’t load the server settings: {{ server.error.value }}
      </Alert>

      <template v-if="server.settings.value">
        <SettingsSection
          title="Project discovery"
          hint="Saved on the server, shared by every browser"
        >
          <SettingsRow
            label="Root directories"
            :hint="
              projects.roots.length
                ? `${projects.roots.length} ${projects.roots.length === 1 ? 'root' : 'roots'} scanned`
                : 'None yet'
            "
          >
            <RouterLink
              to="/roots"
              class="text-[12.5px] text-accent-text hover:underline"
            >
              Manage roots →
            </RouterLink>
          </SettingsRow>
          <SettingsRow
            label="Scan depth"
            :hint="`How many folder levels below a root are searched (1 to ${server.settings.value.max_scan_depth}). Changing it rescans.`"
            for="scan-depth"
          >
            <ScanDepthField
              id="scan-depth"
              :value="server.settings.value.scan_depth"
              :max="server.settings.value.max_scan_depth"
              @commit="saveDepth"
            />
          </SettingsRow>
        </SettingsSection>

        <SettingsSection title="Docker" hint="Saved on the server">
          <SettingsRow
            label="Context on launch"
            hint="Used when DockDuck starts without a remembered choice."
            for="default-context"
          >
            <select
              id="default-context"
              class="h-7 rounded-md border border-line-strong bg-app px-2 text-[12.5px] text-fg"
              :value="server.settings.value.default_context"
              @change="
                server.save({
                  default_context: ($event.target as HTMLSelectElement).value,
                })
              "
            >
              <option value="">last used</option>
              <option v-for="n in contextOptions" :key="n" :value="n">
                {{ n }}
              </option>
            </select>
          </SettingsRow>
          <div v-if="savedContextMissing" class="px-3.5 py-2">
            <Alert tone="warn">
              Docker no longer has a context named
              <b class="font-mono">{{
                server.settings.value.default_context
              }}</b>; DockDuck falls back to the last used one.
            </Alert>
          </div>
          <SettingsRow
            label="Also switch the docker CLI"
            hint="Run `docker context use` when you switch here, so the terminal agrees. Off by default: DockDuck never changes your CLI on its own."
          >
            <Switch
              label="Also switch the docker CLI"
              :model-value="server.settings.value.sync_cli"
              @update:model-value="server.save({ sync_cli: $event })"
            />
          </SettingsRow>
          <SettingsRow label="Compose binary">
            <span class="font-mono text-[12.5px] text-muted">
              {{
                server.settings.value.compose_version
                  ? `Docker Compose v${server.settings.value.compose_version}`
                  : "not detected"
              }}
            </span>
          </SettingsRow>
        </SettingsSection>
      </template>

      <SettingsSection
        title="Terminal"
        hint="Defaults for new sessions, stored in this browser"
      >
        <SettingsRow
          label="Default shell"
          hint="Empty picks bash and falls back to sh."
          for="term-shell"
        >
          <input
            id="term-shell"
            v-model="terminals.prefs.shell"
            placeholder="/bin/bash"
            spellcheck="false"
            class="h-7 w-44 rounded-md border border-line-strong bg-app px-2 font-mono text-[12.5px] text-fg"
          />
        </SettingsRow>
        <SettingsRow
          label="Default user"
          hint="Pre-selected when you open a session."
        >
          <select
            v-model="terminals.prefs.user"
            aria-label="Default user"
            class="h-7 rounded-md border border-line-strong bg-app px-2 text-[12.5px] text-fg"
          >
            <option v-for="o in userOptions" :key="o.value" :value="o.value">
              {{ o.label }}
            </option>
          </select>
          <input
            v-if="terminals.prefs.user === 'custom'"
            v-model="terminals.prefs.customUser"
            aria-label="Custom user"
            placeholder="name or UID"
            spellcheck="false"
            class="h-7 w-28 rounded-md border border-line-strong bg-app px-2 font-mono text-[12.5px] text-fg"
          />
        </SettingsRow>
        <SettingsRow label="Font size" for="term-font">
          <input
            id="term-font"
            v-model.number="terminals.fontSize"
            type="number"
            min="9"
            max="24"
            class="h-7 w-16 rounded-md border border-line-strong bg-app px-2 text-center text-[13px] text-fg"
          />
        </SettingsRow>
      </SettingsSection>

      <SettingsSection title="Behavior" hint="Stored in this browser only">
        <SettingsRow
          label="Confirm before compose down"
          hint="Ask before removing containers and networks. Deleting volumes always asks, and you type the project name."
        >
          <Switch
            v-model="prefs.confirmDown"
            label="Confirm before compose down"
          />
        </SettingsRow>
        <SettingsRow
          label="Toasts for finished operations"
          hint="Up, down, restart and pull results. Failures are always shown."
        >
          <Switch
            v-model="prefs.doneToasts"
            label="Toasts for finished operations"
          />
        </SettingsRow>
      </SettingsSection>

      <AccountSection />
    </div>
  </div>
</template>
