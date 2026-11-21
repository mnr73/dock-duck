import { computed, onBeforeUnmount, ref, watch } from "vue";

import { onBeforeRouteLeave } from "vue-router";

import { useEventListener } from "@vueuse/core";

import { ApiError, errorMessage } from "@/api/client";
import { filesApi } from "@/api/files";
import type { FileContent, Problem } from "@/api/types";

/**
 * State machine shared by the Compose and .env editors:
 * load -> edit (dirty) -> [validate] -> save (ETag-guarded) -> loaded again.
 * A save is refused by the server (409) when the file changed on disk; `conflict`
 * then offers "reload from disk" instead of silently overwriting someone else's edit.
 */
export function useFileEditor(
    project: () => string,
    file: () => string,
    opts: { validate: boolean; reveal?: () => boolean },
) {
    const loaded = ref<FileContent>();
    const text = ref("");
    const loading = ref(false);
    const saving = ref(false);
    const validating = ref(false);
    const error = ref("");
    const conflict = ref(false);
    const problems = ref<Problem[]>([]);
    const savedAt = ref<Date>();
    /** which text the `problems` were computed for: stale results are not shown as current */
    const validatedText = ref("");

    const dirty = computed(
        () => !!loaded.value && text.value !== loaded.value.content,
    );
    const hasErrors = computed(() =>
        problems.value.some((p) => p.severity === "error"),
    );
    const problemsStale = computed(
        () => problems.value.length > 0 && validatedText.value !== text.value,
    );

    let loadSeq = 0;
    async function load(reveal = false) {
        const seq = ++loadSeq;
        loading.value = true;
        error.value = "";
        conflict.value = false;
        try {
            const c = await filesApi.read(project(), file(), reveal);
            if (seq !== loadSeq) return; // a newer load superseded this one
            loaded.value = c;
            text.value = c.content;
            problems.value = [];
            if (opts.validate) await validate();
        } catch (e) {
            if (seq === loadSeq) error.value = errorMessage(e);
        } finally {
            if (seq === loadSeq) loading.value = false;
        }
    }

    async function validate(): Promise<boolean> {
        if (!opts.validate) return true;
        const body = text.value;
        validating.value = true;
        try {
            const v = await filesApi.validate(project(), file(), body);
            problems.value = v.problems;
            validatedText.value = body;
            return v.ok;
        } catch (e) {
            error.value = errorMessage(e);
            return false;
        } finally {
            validating.value = false;
        }
    }

    /** Returns true when the file was written. Invalid Compose is refused unless `force`. */
    async function save(force = false): Promise<boolean> {
        if (!loaded.value || saving.value) return false;
        error.value = "";
        if (opts.validate && !force) {
            const ok = await validate();
            if (!ok) return false;
        }
        saving.value = true;
        try {
            const r = await filesApi.save(
                project(),
                file(),
                text.value,
                loaded.value.etag,
            );
            loaded.value = {
                ...loaded.value,
                content: text.value,
                etag: r.etag,
                exists: true,
            };
            savedAt.value = new Date(r.saved_at);
            conflict.value = false;
            return true;
        } catch (e) {
            if (e instanceof ApiError && e.status === 409)
                conflict.value = true;
            else error.value = errorMessage(e);
            return false;
        } finally {
            saving.value = false;
        }
    }

    function discard() {
        if (loaded.value) text.value = loaded.value.content;
        problems.value = [];
        error.value = "";
    }

    // reload when switching file or project
    watch([project, file], () => load(opts.reveal?.() ?? false), {
        immediate: true,
    });

    // closing the tab with unsaved changes
    useEventListener(window, "beforeunload", (e) => {
        if (dirty.value) e.preventDefault();
    });
    onBeforeRouteLeave(() => {
        if (
            dirty.value &&
            !window.confirm("You have unsaved changes. Leave without saving?")
        )
            return false;
    });
    onBeforeUnmount(() => {
        loadSeq++;
    });

    return {
        loaded,
        text,
        loading,
        saving,
        validating,
        error,
        conflict,
        problems,
        problemsStale,
        savedAt,
        dirty,
        hasErrors,
        load,
        validate,
        save,
        discard,
    };
}
