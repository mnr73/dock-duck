<script setup lang="ts">
import { RouterView } from "vue-router";

import { useColorMode } from "@vueuse/core";

// "auto" follows the OS; the class "dark" is toggled on <html>.
useColorMode({
  selector: "html",
  attribute: "class",
  modes: { light: "light", dark: "dark" },
});
</script>

<template>
  <RouterView />
</template>
