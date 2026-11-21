import { computed, ref } from "vue";

import { defineStore } from "pinia";

import { authApi } from "@/api/auth";
import {
    ApiError,
    errorMessage,
    setCsrfToken,
    setUnauthorizedHandler,
} from "@/api/client";
import type { AuthStatus } from "@/api/types";

/**
 * Login state. `ready` is false until the first /auth/status answer, so the app
 * never flashes the UI (or the login form) before it knows which one to show.
 */
export const useAuthStore = defineStore("auth", () => {
    const status = ref<AuthStatus | null>(null);
    const ready = ref(false);
    const unreachable = ref(false);
    const busy = ref(false);
    const error = ref("");
    /** seconds until another login attempt is allowed (a lock after repeated failures) */
    const lockedFor = ref(0);
    let lockTimer: ReturnType<typeof setInterval> | undefined;

    const authenticated = computed(() => !!status.value?.authenticated);
    /** a password protects this server (as opposed to open mode on a loopback-only install) */
    const protectedMode = computed(() => !!status.value?.configured);
    const needsSetup = computed(
        () =>
            !!status.value && !status.value.configured && status.value.required,
    );

    function apply(s: AuthStatus) {
        status.value = s;
        setCsrfToken(s.csrf ?? "");
    }

    async function load() {
        try {
            apply(await authApi.status());
            unreachable.value = false;
        } catch (e) {
            unreachable.value = e instanceof ApiError && e.code === "network";
        } finally {
            ready.value = true;
        }
    }

    function startLock(seconds: number) {
        lockedFor.value = Math.max(1, Math.ceil(seconds));
        clearInterval(lockTimer);
        lockTimer = setInterval(() => {
            lockedFor.value -= 1;
            if (lockedFor.value <= 0) {
                clearInterval(lockTimer);
                lockedFor.value = 0;
                error.value = "";
            }
        }, 1000);
    }

    async function run(fn: () => Promise<{ csrf: string }>): Promise<boolean> {
        busy.value = true;
        error.value = "";
        try {
            const r = await fn();
            setCsrfToken(r.csrf);
            await load();
            return true;
        } catch (e) {
            error.value = errorMessage(e);
            if (
                e instanceof ApiError &&
                (e.code === "locked" || e.code === "busy")
            )
                startLock(e.retryAfter || 5);
            return false;
        } finally {
            busy.value = false;
        }
    }

    const login = (pw: string) => run(() => authApi.login(pw));
    const setup = (pw: string) => run(() => authApi.setup(pw));
    const changePassword = (current: string, next: string) =>
        run(() => authApi.changePassword(current, next));

    async function logout() {
        try {
            await authApi.logout();
        } catch {
            /* the session may already be gone */
        }
        setCsrfToken("");
        await load();
        // do not leave the previous user's data in memory
        location.assign("/");
    }

    // any 401 from the API means the session ended (logout elsewhere, password changed, expiry)
    setUnauthorizedHandler(() => {
        if (status.value?.authenticated) {
            status.value = {
                ...status.value,
                authenticated: false,
                csrf: undefined,
            };
            setCsrfToken("");
            void load();
        }
    });

    return {
        status,
        ready,
        unreachable,
        busy,
        error,
        lockedFor,
        authenticated,
        protectedMode,
        needsSetup,
        load,
        login,
        setup,
        changePassword,
        logout,
    };
});
