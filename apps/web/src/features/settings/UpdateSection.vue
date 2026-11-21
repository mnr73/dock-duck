<script setup lang="ts">
import { computed, ref } from "vue";

import { IconDownload, IconRefresh } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";

import { errorMessage } from "@/api/client";
import type { UpdateStatus } from "@/api/types";
import { updateApi } from "@/api/update";
import { useSystemStore } from "@/stores/system";

import SettingsRow from "./SettingsRow.vue";
import SettingsSection from "./SettingsSection.vue";

const system = useSystemStore();

const status = ref<UpdateStatus | null>(null);
const checking = ref(false);
const installing = ref(false);
const error = ref("");
/** after install: waiting for the server to come back with the new version */
const restarting = ref(false);
const done = ref("");

const isDev = computed(() => !/^v?\d+\.\d+\.\d+/.test(system.version));

async function check() {
  checking.value = true;
  error.value = "";
  try {
    status.value = await updateApi.status(true);
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    checking.value = false;
  }
}

/** Polls /health until the new version answers; the server restarts itself. */
async function waitForRestart(expected: string) {
  restarting.value = true;
  const until = Date.now() + 60_000;
  await new Promise((r) => setTimeout(r, 1500));
  while (Date.now() < until) {
    try {
      await system.load();
      if (!system.serverDown && system.version === expected) {
        restarting.value = false;
        done.value = expected;
        status.value = null;
        return;
      }
    } catch {
      /* still restarting */
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  restarting.value = false;
  error.value =
    "The update is installed but DockDuck did not come back in time. Start it again (or `systemctl restart dockduck`).";
}

async function install() {
  if (!status.value?.latest) return;
  installing.value = true;
  error.value = "";
  const target = status.value.latest;
  try {
    await updateApi.install();
  } catch (e) {
    error.value = errorMessage(e);
    installing.value = false;
    return;
  }
  installing.value = false;
  await waitForRestart(target);
}
</script>

<template>
  <SettingsSection
    title="Updates"
    hint="Checked only when you ask. DockDuck never contacts GitHub on its own."
  >
    <SettingsRow
      label="Version"
      :hint="
        isDev
          ? 'A development build: it cannot compare or update itself.'
          : 'Releases come from GitHub.'
      "
    >
      <span class="font-mono text-[12.5px] text-muted">{{
        system.version || "unknown"
      }}</span>
      <Btn size="sm" :loading="checking" :disabled="restarting" @click="check">
        <IconRefresh class="size-3.5" /> Check for updates
      </Btn>
    </SettingsRow>

    <div v-if="error" class="px-3.5 py-3">
      <Alert tone="err">{{ error }}</Alert>
    </div>
    <div v-if="done" class="px-3.5 py-3">
      <Alert tone="ok">Updated to {{ done }}.</Alert>
    </div>
    <div v-if="restarting" class="px-3.5 py-3">
      <Alert tone="info">
        Installed. Restarting DockDuck… open terminals are closed by the
        restart.
      </Alert>
    </div>

    <div v-if="status && !status.installed && !restarting" class="px-3.5 py-3">
      <Alert v-if="!status.available" tone="ok">
        You are on the latest version ({{ status.latest }}).
      </Alert>
      <div v-else class="flex flex-col gap-3">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-[13px]">
              <b>{{ status.latest }}</b> is available
              <a
                v-if="status.url"
                :href="status.url"
                target="_blank"
                rel="noopener noreferrer"
                class="text-accent-text hover:underline"
                >release notes</a
              >
            </p>
            <p v-if="!status.can_install" class="text-[12px] text-warn-fg">
              {{ status.reason }}
            </p>
            <p v-else class="text-[12px] text-muted">
              {{
                status.managed
                  ? "The service restarts by itself after the update."
                  : "DockDuck restarts itself after the update."
              }}
              Terminals that are open will be closed.
            </p>
          </div>
          <Btn
            variant="primary"
            :loading="installing"
            :disabled="!status.can_install"
            @click="install"
          >
            <IconDownload class="size-3.5" /> Update to {{ status.latest }}
          </Btn>
        </div>
        <pre
          v-if="status.notes"
          class="max-h-40 overflow-auto rounded-md border border-line bg-raised p-2.5 text-[12px] whitespace-pre-wrap text-muted"
          >{{ status.notes }}</pre>
      </div>
    </div>
  </SettingsSection>
</template>
