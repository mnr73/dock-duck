<script setup lang="ts">
import { computed } from "vue";

import { usePrefsStore } from "@/stores/prefs";
import { brandSvg } from "@/utils/brand";

const prefs = usePrefsStore();
/** the supplied duck, recoloured with the chosen accent (a trusted local asset) */
const svg = computed(() => brandSvg(prefs.accent));
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <span role="img" aria-label="DockDuck" class="inline-block" v-html="svg" />
</template>

<style scoped>
span :deep(svg) {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
