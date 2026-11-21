<script setup lang="ts">
import { computed, ref, watch } from "vue";

import { IconOpenWindow, IconTerminal } from "@/icons";

import Alert from "@/components/ui/Alert.vue";
import Btn from "@/components/ui/Btn.vue";
import Dialog from "@/components/ui/Dialog.vue";

import type {
  ServiceInfo,
  SystemTerminalStatus,
  TerminalOpenRequest,
} from "@/api/types";
import { type UserChoice, useTerminalsStore } from "@/stores/terminals";

const props = defineProps<{
  services: ServiceInfo[];
  initialService?: string;
  system: SystemTerminalStatus | null;
  busy?: boolean;
  error?: string;
}>();
const open = defineModel<boolean>("open", { required: true });
const emit = defineEmits<{
  submit: [req: TerminalOpenRequest, where: "browser" | "system"];
}>();

const store = useTerminalsStore();

const running = computed(() =>
  props.services.filter((s) => s.containers.some((c) => c.state === "running")),
);
const service = ref("");
const container = ref("");
const command = ref("");
const workdir = ref("");

const replicas = computed(() =>
  (
    props.services.find((s) => s.name === service.value)?.containers ?? []
  ).filter((c) => c.state === "running"),
);

function pickService() {
  const wanted = props.initialService;
  service.value =
    (wanted && running.value.some((s) => s.name === wanted)
      ? wanted
      : (running.value.find((s) => s.name === service.value)?.name ??
        running.value[0]?.name)) ?? "";
}

watch(
  open,
  (o) => {
    if (!o) return;
    container.value = "";
    command.value = "";
    workdir.value = "";
    pickService();
  },
  { immediate: true },
);
// the dialog can open before the project's services have loaded (page just opened)
watch(running, () => {
  if (open.value && !running.value.some((s) => s.name === service.value))
    pickService();
});
watch(service, () => (container.value = ""));

const userOptions: {
  id: UserChoice;
  label: string;
  sub: string;
  dot: string;
}[] = [
  {
    id: "default",
    label: "Container default",
    sub: "the user set by the image or `user:` in Compose",
    dot: "bg-ok",
  },
  {
    id: "root",
    label: "root (UID 0)",
    sub: "full privileges inside the container",
    dot: "bg-err",
  },
  {
    id: "uid1000",
    label: "UID 1000",
    sub: "usually matches your own user, so files you create keep your ownership",
    dot: "bg-info",
  },
  {
    id: "custom",
    label: "Custom",
    sub: "a name or UID, e.g. www-data or 1001:1001",
    dot: "bg-dim",
  },
];

const userArg = computed(() => store.userArg(store.prefs));
const customInvalid = computed(
  () =>
    store.prefs.user === "custom" &&
    !/^[A-Za-z0-9_][A-Za-z0-9_.-]{0,63}(:[A-Za-z0-9_][A-Za-z0-9_.-]{0,63})?$/.test(
      store.prefs.customUser.trim(),
    ),
);
const shellInvalid = computed(
  () =>
    !!store.prefs.shell &&
    !/^(\/|[A-Za-z0-9_])[A-Za-z0-9_.+/-]*$/.test(store.prefs.shell),
);
const workdirInvalid = computed(
  () => !!workdir.value && !workdir.value.startsWith("/"),
);
const invalid = computed(
  () =>
    !service.value ||
    customInvalid.value ||
    shellInvalid.value ||
    workdirInvalid.value,
);

/** The equivalent command, so what happens is never a mystery. */
const preview = computed(() => {
  const parts = ["docker exec -it"];
  if (userArg.value) parts.push(`-u ${userArg.value}`);
  if (workdir.value) parts.push(`-w ${workdir.value}`);
  parts.push(
    container.value || replicas.value[0]?.name || service.value || "<service>",
  );
  if (command.value)
    parts.push(`${store.prefs.shell || "sh"} -c '${command.value}'`);
  else parts.push(store.prefs.shell || "bash");
  return parts.join(" ");
});

function submit(where: "browser" | "system") {
  if (invalid.value) return;
  emit(
    "submit",
    {
      service: service.value,
      container: container.value || undefined,
      user: userArg.value,
      shell: store.prefs.shell || undefined,
      command: command.value.trim() || undefined,
      workdir: workdir.value || undefined,
    },
    where,
  );
}
</script>

<template>
  <Dialog v-model:open="open" title="New terminal" width="520px">
    <div class="flex flex-col gap-4">
      <Alert v-if="!running.length" tone="warn"
        >No service of this project is running, so there is nothing to attach
        to. Start it first.</Alert
      >

      <div class="grid grid-cols-[1fr_auto] gap-3">
        <label class="flex flex-col gap-1 text-[12px] text-muted">
          Service
          <select
            v-model="service"
            class="h-8 rounded-md border border-line-strong bg-surface px-2 text-[12.5px] text-fg"
          >
            <option v-if="!service" value="" disabled>Select a service…</option>
            <option v-for="s in running" :key="s.name" :value="s.name">
              {{ s.name }}
            </option>
          </select>
        </label>
        <label
          v-if="replicas.length > 1"
          class="flex flex-col gap-1 text-[12px] text-muted"
        >
          Replica
          <select
            v-model="container"
            class="h-8 rounded-md border border-line-strong bg-surface px-2 text-[12.5px] text-fg"
          >
            <option value="">First running</option>
            <option v-for="c in replicas" :key="c.name" :value="c.name">
              {{ c.name }}
            </option>
          </select>
        </label>
      </div>

      <fieldset class="flex flex-col gap-1.5">
        <legend class="mb-1 text-[12px] text-muted">Run as</legend>
        <label
          v-for="u in userOptions"
          :key="u.id"
          class="flex cursor-pointer items-start gap-2.5 rounded-md border px-3 py-2"
          :class="
            store.prefs.user === u.id
              ? 'border-accent bg-accent-soft'
              : 'border-line hover:bg-hover'
          "
        >
          <input
            v-model="store.prefs.user"
            type="radio"
            name="exec-user"
            :value="u.id"
            class="mt-1"
          />
          <span class="mt-1.5 size-1.5 shrink-0 rounded-full" :class="u.dot" />
          <span class="min-w-0 flex-1">
            <span class="block text-[12.5px] text-fg">{{ u.label }}</span>
            <span class="block text-[11.5px] text-muted">{{ u.sub }}</span>
            <input
              v-if="u.id === 'custom' && store.prefs.user === 'custom'"
              v-model="store.prefs.customUser"
              class="mt-1.5 h-7 w-full rounded border bg-surface px-2 font-mono text-[12px]"
              :class="customInvalid ? 'border-err' : 'border-line-strong'"
              placeholder="www-data"
              spellcheck="false"
              autocomplete="off"
            />
          </span>
        </label>
      </fieldset>

      <div class="grid grid-cols-2 gap-3">
        <label class="flex flex-col gap-1 text-[12px] text-muted">
          Shell
          <input
            v-model="store.prefs.shell"
            class="h-8 rounded-md border bg-surface px-2 font-mono text-[12.5px] text-fg"
            :class="shellInvalid ? 'border-err' : 'border-line-strong'"
            placeholder="auto: bash, then sh"
            list="exec-shells"
            spellcheck="false"
          />
          <datalist id="exec-shells">
            <option value="/bin/bash" />
            <option value="/bin/sh" />
            <option value="/bin/zsh" />
            <option value="/bin/ash" />
          </datalist>
        </label>
        <label class="flex flex-col gap-1 text-[12px] text-muted">
          Working directory
          <input
            v-model="workdir"
            class="h-8 rounded-md border bg-surface px-2 font-mono text-[12.5px] text-fg"
            :class="workdirInvalid ? 'border-err' : 'border-line-strong'"
            placeholder="image default"
            spellcheck="false"
          />
        </label>
      </div>

      <label class="flex flex-col gap-1 text-[12px] text-muted">
        Command
        <span class="text-dim">(optional: run this instead of a shell)</span>
        <input
          v-model="command"
          class="h-8 rounded-md border border-line-strong bg-surface px-2 font-mono text-[12.5px] text-fg"
          placeholder="mysql -uerp -p"
          spellcheck="false"
        />
      </label>

      <p
        class="rounded-md bg-raised px-3 py-2 font-mono text-[11.5px] break-all text-muted"
      >
        $ {{ preview }}
      </p>
      <Alert v-if="error" tone="err">{{ error }}</Alert>
      <p v-if="!service && running.length" class="text-[12px] text-muted">
        Choose a service to continue.
      </p>
    </div>

    <template #footer>
      <Btn @click="open = false">Cancel</Btn>
      <Btn
        v-if="system?.available"
        :disabled="invalid || busy"
        :title="`Open in ${system.name} instead of the browser`"
        @click="submit('system')"
      >
        <IconOpenWindow class="size-3.5" /> Open in {{ system.name }}
      </Btn>
      <Btn
        variant="primary"
        :disabled="invalid || busy"
        :loading="busy"
        data-autofocus
        @click="submit('browser')"
      >
        <IconTerminal class="size-3.5" /> Open here
      </Btn>
    </template>
  </Dialog>
</template>
