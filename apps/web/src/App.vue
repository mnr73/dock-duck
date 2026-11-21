<script setup lang="ts">
import { onMounted } from "vue";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import Toaster from "@/components/ui/Toaster.vue";

import LoginScreen from "@/features/auth/LoginScreen.vue";
import AppShell from "@/features/layout/AppShell.vue";

import { useAppearance } from "@/composables/useAppearance";
import { useTheme } from "@/composables/useTheme";
import { useAuthStore } from "@/stores/auth";

useTheme(); // applies the saved/system theme class on <html>
useAppearance(); // accent colour and density

const auth = useAuthStore();
onMounted(() => auth.load());
</script>

<template>
  <!-- nothing is rendered before the server has said who we are: no flash of the app or of the form -->
  <div v-if="!auth.ready" class="min-h-full bg-app" />
  <div
    v-else-if="auth.unreachable && !auth.status"
    class="grid min-h-full place-items-center bg-app p-6"
  >
    <div class="flex max-w-sm flex-col gap-3">
      <Alert tone="err">DockDuck server is unreachable. Is it running?</Alert>
      <Btn @click="auth.load()">Try again</Btn>
    </div>
  </div>
  <LoginScreen v-else-if="!auth.authenticated" />
  <AppShell v-else />
  <Toaster />
</template>
