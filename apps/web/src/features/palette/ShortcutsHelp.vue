<script setup lang="ts">
import Dialog from "@/components/ui/Dialog.vue";

const open = defineModel<boolean>("open", { required: true });

const groups: { title: string; rows: [string[], string][] }[] = [
  {
    title: "Anywhere",
    rows: [
      [["Ctrl", "K"], "Command palette"],
      [["?"], "This list"],
      [["["], "Collapse or expand the sidebar"],
      [["g", "p"], "Go to Projects"],
      [["g", "o"], "Go to Ports"],
      [["g", "r"], "Go to Registries"],
      [["g", "s"], "Go to Settings"],
    ],
  },
  {
    title: "Logs",
    rows: [
      [["Ctrl", "F"], "Search in the log"],
      [["Esc"], "Clear the selection"],
    ],
  },
  {
    title: "Editors",
    rows: [[["Ctrl", "S"], "Save the compose file or .env"]],
  },
];
</script>

<template>
  <Dialog v-model:open="open" title="Keyboard shortcuts" width="440px">
    <div class="flex flex-col gap-4">
      <section v-for="g in groups" :key="g.title">
        <h3 class="mb-1.5 text-[11px] tracking-wide text-dim uppercase">
          {{ g.title }}
        </h3>
        <ul class="flex flex-col gap-1.5">
          <li
            v-for="[keys, label] in g.rows"
            :key="label"
            class="flex items-center justify-between gap-3"
          >
            <span class="text-[13px]">{{ label }}</span>
            <span class="flex shrink-0 items-center gap-1">
              <template v-for="(k, i) in keys" :key="k">
                <span
                  v-if="i > 0 && keys[0] === 'g'"
                  class="text-[11px] text-dim"
                  >then</span
                >
                <kbd
                  class="min-w-5 rounded border border-line-strong bg-raised px-1.5 py-0.5 text-center font-mono text-[11px]"
                  >{{ k }}</kbd
                >
              </template>
            </span>
          </li>
        </ul>
      </section>
      <p class="text-[12px] text-dim">
        Single keys are ignored while you type in a field, the editor or a
        terminal.
      </p>
    </div>
  </Dialog>
</template>
