import { ref } from "vue";

import { toast } from "vue-sonner";

import { errorMessage } from "@/api/client";
import { settingsApi } from "@/api/settings";
import type { ServerSettings, ServerSettingsPatch } from "@/api/types";

/**
 * Server-side settings with save-as-you-go: `save` sends only what changed, shows
 * the server's own refusal if there is one, and always ends with what the server
 * actually stored, so a rejected value never lingers in the UI.
 */
export function useServerSettings() {
    const settings = ref<ServerSettings | null>(null);
    const loading = ref(false);
    const error = ref("");
    const saving = ref(false);

    async function load() {
        loading.value = true;
        try {
            settings.value = await settingsApi.get();
            error.value = "";
        } catch (e) {
            error.value = errorMessage(e);
        } finally {
            loading.value = false;
        }
    }

    async function save(patch: ServerSettingsPatch): Promise<boolean> {
        saving.value = true;
        try {
            settings.value = await settingsApi.update(patch);
            return true;
        } catch (e) {
            toast.error("Couldn’t save the setting", {
                description: errorMessage(e),
            });
            await load();
            return false;
        } finally {
            saving.value = false;
        }
    }

    return { settings, loading, error, saving, load, save };
}
