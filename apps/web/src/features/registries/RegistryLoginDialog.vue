<script setup lang="ts">
import { computed, ref, watch } from "vue";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";
import SegmentedControl from "@/components/ui/SegmentedControl.vue";

import type { RegistryLoginRequest } from "@/api/types";

const props = defineProps<{
  busy?: boolean;
  error?: string;
  /** pre-fill when re-authenticating an existing registry */
  prefill?: { server: string; username?: string; hub: boolean } | null;
}>();
const open = defineModel<boolean>("open", { required: true });
const emit = defineEmits<{ submit: [req: RegistryLoginRequest] }>();

type Kind = "hub" | "private";
const kind = ref<Kind>("hub");
const server = ref("");
const username = ref("");
const password = ref("");

watch(open, (o) => {
  if (!o) {
    password.value = ""; // never keep a secret around after the dialog closes
    return;
  }
  const p = props.prefill;
  kind.value = !p || p.hub ? "hub" : "private";
  server.value = p && !p.hub ? p.server : "";
  username.value = p?.username ?? "";
  password.value = "";
});

const kinds = [
  { value: "hub" as const, label: "Docker Hub" },
  { value: "private" as const, label: "Private registry" },
];
const valid = computed(
  () =>
    username.value.trim().length > 0 &&
    password.value.length > 0 &&
    (kind.value === "hub" || server.value.trim().length > 0),
);

function submit() {
  if (!valid.value || props.busy) return;
  emit("submit", {
    server: kind.value === "hub" ? "" : server.value.trim(),
    username: username.value.trim(),
    password: password.value,
  });
}
</script>

<template>
  <Dialog v-model:open="open" title="Sign in to a registry" width="460px">
    <form class="flex flex-col gap-3.5" novalidate @submit.prevent="submit">
      <SegmentedControl
        v-model="kind"
        :options="kinds"
        aria-label="Registry type"
      />

      <label
        v-if="kind === 'private'"
        class="flex flex-col gap-1 text-[12px] text-muted"
      >
        Registry address
        <input
          v-model="server"
          class="h-8 rounded-md border border-line-strong bg-app px-2.5 font-mono text-[12.5px] text-fg"
          placeholder="ghcr.io or registry.example.com:5000"
          spellcheck="false"
          autocomplete="off"
        />
      </label>
      <p v-else class="font-mono text-[12px] text-dim">
        https://index.docker.io/v1/
      </p>

      <label class="flex flex-col gap-1 text-[12px] text-muted">
        Username
        <input
          v-model="username"
          class="h-8 rounded-md border border-line-strong bg-app px-2.5 text-[13px] text-fg"
          autocomplete="username"
          spellcheck="false"
        />
      </label>
      <label class="flex flex-col gap-1 text-[12px] text-muted">
        Password or access token
        <input
          v-model="password"
          type="password"
          class="h-8 rounded-md border border-line-strong bg-app px-2.5 text-[13px] text-fg"
          autocomplete="current-password"
          spellcheck="false"
        />
        <span class="text-[11.5px] text-dim">
          Prefer a personal access token with the least scope you need (for
          example read-only pull) over your account password. It is sent to
          Docker directly and never stored by DockDuck.
        </span>
      </label>

      <Alert v-if="error" tone="err">{{ error }}</Alert>
      <button type="submit" class="hidden" />
    </form>
    <template #footer>
      <Btn @click="open = false">Cancel</Btn>
      <Btn variant="primary" :disabled="!valid" :loading="busy" @click="submit"
        >Sign in</Btn
      >
    </template>
  </Dialog>
</template>
