<script setup lang="ts">
import { computed, ref } from "vue";

import Alert from "@/components/ui/Alert.vue";
import BrandMark from "@/components/ui/BrandMark.vue";
import Btn from "@/components/ui/Btn.vue";

import { useAuthStore } from "@/stores/auth";

const auth = useAuthStore();
const password = ref("");
const repeat = ref("");

/** first run on a machine the server can configure from this browser */
const setupHere = computed(
  () => !auth.protectedMode && auth.status?.setup_here,
);
/** a network-facing server without a password cannot be set up from a remote browser */
const setupElsewhere = computed(
  () => !auth.protectedMode && !auth.status?.setup_here,
);

const MIN = 10;
const tooShort = computed(
  () => password.value.length > 0 && password.value.length < MIN,
);
const mismatch = computed(
  () => repeat.value.length > 0 && password.value !== repeat.value,
);
const canSubmit = computed(() =>
  setupHere.value
    ? password.value.length >= MIN && password.value === repeat.value
    : password.value.length > 0,
);

async function submit() {
  if (!canSubmit.value || auth.busy || auth.lockedFor > 0) return;
  const ok = setupHere.value
    ? await auth.setup(password.value)
    : await auth.login(password.value);
  password.value = "";
  repeat.value = "";
  if (!ok) document.getElementById("dd-password")?.focus();
}
</script>

<template>
  <div class="grid min-h-full place-items-center bg-app p-6">
    <form
      class="w-full max-w-[380px] rounded-xl border border-line-strong bg-surface p-6 shadow-pop"
      novalidate
      @submit.prevent="submit"
    >
      <div class="mb-5 flex items-center gap-2.5">
        <BrandMark class="size-10" />
        <div>
          <h1 class="text-[16px] leading-tight font-semibold tracking-tight">
            DockDuck
          </h1>
          <p class="text-[12px] text-muted">
            {{ setupHere ? "Choose a password" : "Sign in to continue" }}
          </p>
        </div>
      </div>

      <Alert v-if="setupElsewhere" tone="warn">
        No password is set on this server, so it cannot be used from the network
        yet. Run <code class="font-mono">dockduck passwd</code> on the server,
        then reload this page.
      </Alert>

      <template v-else>
        <p v-if="setupHere" class="mb-3 text-[12.5px] text-muted">
          DockDuck can control every container on this machine, so it asks for a
          password. Use at least {{ MIN }} characters; a few random words work
          well.
        </p>

        <label class="mb-3 flex flex-col gap-1 text-[12px] text-muted">
          Password
          <input
            id="dd-password"
            v-model="password"
            type="password"
            class="h-9 rounded-md border bg-app px-3 text-[13px] text-fg"
            :class="tooShort && setupHere ? 'border-err' : 'border-line-strong'"
            :autocomplete="setupHere ? 'new-password' : 'current-password'"
            autofocus
            spellcheck="false"
          />
          <span v-if="tooShort && setupHere" class="text-[11.5px] text-err-fg">
            {{ MIN - password.length }} more
            {{ MIN - password.length === 1 ? "character" : "characters" }}
          </span>
        </label>

        <label
          v-if="setupHere"
          class="mb-3 flex flex-col gap-1 text-[12px] text-muted"
        >
          Repeat the password
          <input
            v-model="repeat"
            type="password"
            class="h-9 rounded-md border bg-app px-3 text-[13px] text-fg"
            :class="mismatch ? 'border-err' : 'border-line-strong'"
            autocomplete="new-password"
            spellcheck="false"
          />
          <span v-if="mismatch" class="text-[11.5px] text-err-fg"
            >The two passwords differ.</span
          >
        </label>

        <Alert v-if="auth.error" tone="err" class="mb-3">
          {{ auth.error }}
          <span
            v-if="auth.lockedFor > 0"
            class="mt-0.5 block text-[12px] text-muted"
          >
            You can try again in {{ auth.lockedFor }} s.
          </span>
        </Alert>

        <Btn
          type="submit"
          variant="primary"
          class="w-full"
          :loading="auth.busy"
          :disabled="!canSubmit || auth.lockedFor > 0"
        >
          {{ setupHere ? "Set password" : "Sign in" }}
        </Btn>

        <p v-if="!setupHere" class="mt-4 text-[11.5px] text-dim">
          Forgot it? Run <code class="font-mono">dockduck passwd</code> on the
          server.
        </p>
      </template>
    </form>
  </div>
</template>
