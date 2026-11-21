import { createApp } from "vue";

import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
import { createPinia } from "pinia";

import "@/styles/main.css";

import App from "./App.vue";
import { router } from "./router";

createApp(App).use(createPinia()).use(router).mount("#app");
