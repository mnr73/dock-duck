<script setup lang="ts">
import { computed, ref } from "vue";

import { toast } from "vue-sonner";

import { IconKey, IconLogout, IconShield } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";

import { useAuthStore } from "@/stores/auth";

const auth = useAuthStore();
const open = ref(false);
const current = ref("");
const next = ref("");
const repeat = ref("");
const MIN = 10;

/** open mode: loopback-only install without a password yet */
const unprotected = computed(() => !auth.protectedMode);
const valid = computed(
  () =>
    next.value.length >= MIN &&
    next.value === repeat.value &&
    (unprotected.value || current.value.length > 0),
);

function show() {
  current.value = next.value = repeat.value = "";
  auth.error = "";
  open.value = true;
}

async function submit() {
  if (!valid.value || auth.busy) return;
  // read it before the call: a successful setup flips `unprotected`, and the
  // message must describe what the user just did
  const firstTime = unprotected.value;
  const ok = firstTime
    ? await auth.setup(next.value)
    : await auth.changePassword(current.value, next.value);
  if (ok) {
    open.value = false;
    toast.success(
      firstTime
        ? "Password set. You are signed in."
        : "Password changed; other sessions were signed out",
    );
  }
}
</script>

<template>
  <section
    class="flex items-center justify-between gap-4 rounded-[9px] border border-line p-3.5"
  >
    <div class="flex min-w-0 items-start gap-3">
      <IconShield
        class="mt-0.5 size-4 shrink-0"
        :class="unprotected ? 'text-warn' : 'text-ok'"
      />
      <div>
        <h2 class="font-medium">Login</h2>
        <p v-if="unprotected" class="text-[12px] text-muted">
          No password is set: anyone who can open this page on this machine
          controls your containers.
        </p>
        <p v-else class="text-[12px] text-muted">
          Password protected. Sessions end after 12 hours without use, or 7 days
          at most.
        </p>
      </div>
    </div>
    <div class="flex shrink-0 gap-2">
      <Btn :variant="unprotected ? 'primary' : 'secondary'" @click="show">
        <IconKey class="size-3.5" />
        {{ unprotected ? "Set password" : "Change password" }}
      </Btn>
      <Btn v-if="!unprotected" @click="auth.logout()"
        ><IconLogout class="size-3.5" /> Sign out</Btn
      >
    </div>
  </section>

  <Dialog
    v-model:open="open"
    :title="unprotected ? 'Set a password' : 'Change password'"
    width="420px"
  >
    <form class="flex flex-col gap-3" novalidate @submit.prevent="submit">
      <label
        v-if="!unprotected"
        class="flex flex-col gap-1 text-[12px] text-muted"
      >
        Current password
        <input
          v-model="current"
          type="password"
          class="h-8 rounded-md border border-line-strong bg-app px-2.5 text-[13px] text-fg"
          autocomplete="current-password"
        />
      </label>
      <label class="flex flex-col gap-1 text-[12px] text-muted">
        New password
        <span class="text-dim">(at least {{ MIN }} characters)</span>
        <input
          v-model="next"
          type="password"
          class="h-8 rounded-md border border-line-strong bg-app px-2.5 text-[13px] text-fg"
          autocomplete="new-password"
        />
      </label>
      <label class="flex flex-col gap-1 text-[12px] text-muted">
        Repeat the new password
        <input
          v-model="repeat"
          type="password"
          class="h-8 rounded-md border bg-app px-2.5 text-[13px] text-fg"
          :class="
            repeat && repeat !== next ? 'border-err' : 'border-line-strong'
          "
          autocomplete="new-password"
        />
      </label>
      <Alert v-if="auth.error" tone="err">{{ auth.error }}</Alert>
      <button type="submit" class="hidden" />
    </form>
    <template #footer>
      <Btn @click="open = false">Cancel</Btn>
      <Btn
        variant="primary"
        :disabled="!valid"
        :loading="auth.busy"
        @click="submit"
      >
        {{ unprotected ? "Set password" : "Change password" }}
      </Btn>
    </template>
  </Dialog>
</template>
