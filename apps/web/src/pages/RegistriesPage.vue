<script setup lang="ts">
import { onMounted, ref } from "vue";

import { toast } from "vue-sonner";

import { IconAdd, IconKey, IconLogout, IconRefresh, IconSignIn } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Badge from "@/components/ui/Badge.vue";
import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";
import EmptyState from "@/components/ui/EmptyState.vue";
import Spinner from "@/components/ui/Spinner.vue";

import PageHeader from "@/features/layout/PageHeader.vue";
import RegistryLoginDialog from "@/features/registries/RegistryLoginDialog.vue";

import { errorMessage } from "@/api/client";
import { registriesApi } from "@/api/registries";
import type {
  RegistryEntry,
  RegistryLoginRequest,
  RegistryState,
} from "@/api/types";

const state = ref<RegistryState | null>(null);
const loading = ref(false);
const loadError = ref("");

const dialog = ref(false);
const prefill = ref<{ server: string; username?: string; hub: boolean } | null>(
  null,
);
const busy = ref(false);
const loginError = ref("");

const signOut = ref<RegistryEntry | null>(null);
const signingOut = ref(false);
const signOutOpen = ref(false);

async function load() {
  loading.value = true;
  loadError.value = "";
  try {
    state.value = await registriesApi.state();
  } catch (e) {
    loadError.value = errorMessage(e);
  } finally {
    loading.value = false;
  }
}
onMounted(load);

function openLogin(entry?: RegistryEntry) {
  prefill.value = entry
    ? { server: entry.server, username: entry.username, hub: entry.hub }
    : null;
  loginError.value = "";
  dialog.value = true;
}

async function login(req: RegistryLoginRequest) {
  busy.value = true;
  loginError.value = "";
  try {
    const r = await registriesApi.login(req);
    dialog.value = false;
    toast.success(`Signed in to ${req.server || "Docker Hub"}`);
    if (r.plaintext)
      toast.warning("Docker stored the credentials unencrypted in config.json");
    await load();
  } catch (e) {
    loginError.value = errorMessage(e);
  } finally {
    busy.value = false;
  }
}

function askSignOut(e: RegistryEntry) {
  signOut.value = e;
  signOutOpen.value = true;
}

async function confirmSignOut() {
  if (!signOut.value) return;
  signingOut.value = true;
  try {
    await registriesApi.logout(signOut.value.hub ? "" : signOut.value.server);
    toast.success(`Signed out of ${signOut.value.name}`);
    signOutOpen.value = false;
    await load();
  } catch (e) {
    toast.error(errorMessage(e));
  } finally {
    signingOut.value = false;
  }
}
</script>

<template>
  <div class="flex min-h-full flex-col">
    <PageHeader
      title="Registries"
      subtitle="Where Docker pulls and pushes images as you"
    >
      <template #actions>
        <Btn size="sm" :loading="loading" @click="load"
          ><IconRefresh class="size-3.5" /> Refresh</Btn
        >
        <Btn size="sm" variant="primary" @click="openLogin()"
          ><IconAdd class="size-3.5" /> Sign in to a registry</Btn
        >
      </template>
    </PageHeader>

    <div class="flex max-w-[860px] flex-col gap-4 p-5">
      <Alert v-if="loadError" tone="err">{{ loadError }}</Alert>

      <Alert v-if="state?.plaintext" tone="warn">
        Docker stores these credentials <b>unencrypted</b> in
        <span class="font-mono">{{ state.config_path }}</span>. Anyone who can
        read that file can use them. A credential helper (for example
        <span class="font-mono">docker-credential-secretservice</span> or
        <span class="font-mono">pass</span>) keeps them in your system keychain
        instead.
      </Alert>
      <p v-else-if="state" class="text-[12px] text-muted">
        Credentials are kept by
        <span class="font-mono">{{ state.store }}</span>, not in plain text.
      </p>

      <div
        v-if="loading && !state"
        class="flex justify-center py-10 text-muted"
      >
        <Spinner class="size-4" />
      </div>

      <EmptyState
        v-else-if="state && !state.entries.length"
        title="Not signed in anywhere"
        text="Public images need no sign-in. Sign in to pull private images or push your own."
      >
        <template #icon><IconKey class="size-5" /></template>
        <Btn variant="primary" @click="openLogin()"
          ><IconSignIn class="size-3.5" /> Sign in to a registry</Btn
        >
      </EmptyState>

      <ul
        v-else-if="state"
        class="divide-y divide-line overflow-hidden rounded-[9px] border border-line"
      >
        <li
          v-for="e in state.entries"
          :key="e.server"
          class="flex items-center gap-3 px-3.5 py-3"
        >
          <div
            class="grid size-8 shrink-0 place-items-center rounded-md border border-line-strong bg-raised font-mono text-[11px] text-muted"
          >
            {{ e.name.slice(0, 2).toUpperCase() }}
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate font-medium">{{ e.name }}</span>
              <Badge tone="ok">Signed in</Badge>
              <Badge v-if="e.store === 'file'" tone="warn">plain text</Badge>
            </div>
            <p class="truncate font-mono text-[11.5px] text-muted">
              {{ e.server
              }}<template v-if="e.username"> · {{ e.username }}</template>
            </p>
          </div>
          <Btn size="sm" @click="openLogin(e)"
            ><IconKey class="size-3.5" /> Re-authenticate</Btn
          >
          <Btn size="sm" variant="danger" @click="askSignOut(e)"
            ><IconLogout class="size-3.5" /> Sign out</Btn
          >
        </li>
      </ul>
    </div>

    <RegistryLoginDialog
      v-model:open="dialog"
      :busy="busy"
      :error="loginError"
      :prefill="prefill"
      @submit="login"
    />

    <Dialog
      v-model:open="signOutOpen"
      :title="`Sign out of ${signOut?.name ?? ''}?`"
      width="420px"
    >
      <p>
        Docker forgets the stored credentials for
        <b class="font-mono text-[12.5px]">{{ signOut?.server }}</b>. Pulling
        private images from it fails until you sign in again.
      </p>
      <template #footer>
        <Btn @click="signOutOpen = false">Cancel</Btn>
        <Btn
          variant="danger"
          :loading="signingOut"
          data-autofocus
          @click="confirmSignOut"
          >Sign out</Btn
        >
      </template>
    </Dialog>
  </div>
</template>
