import {
    type MaybeRefOrGetter,
    computed,
    ref,
    shallowRef,
    toValue,
    watch,
} from "vue";

import { useWebSocket } from "@vueuse/core";

import { type LogLine, type RawLine, toLogLine } from "@/utils/logs";

/** Lines kept in memory; the oldest are dropped first. */
export const MAX_LINES = 20000;

export interface LogStreamOptions {
    projectId: MaybeRefOrGetter<string>;
    /** empty = all services */
    services: MaybeRefOrGetter<string[]>;
    tail: MaybeRefOrGetter<number | "all">;
    /** docker --since value or "" */
    since: MaybeRefOrGetter<string>;
    context?: MaybeRefOrGetter<string | undefined>;
}

/**
 * Live log stream over WebSocket (VueUse useWebSocket).
 *
 * - `lines` is a bounded buffer; `paused` stops publishing but keeps collecting
 *   (`buffered` counts them) and `resume()` flushes.
 * - Changing project/services/tail/since/context restarts the stream with a
 *   fresh buffer.
 * - The server sends `{"type":"end"}` when docker's process ends (project
 *   removed, daemon gone); the stream then shows `ended` and does not reconnect
 *   by itself until `restart()`.
 */
export function useLogStream(o: LogStreamOptions) {
    const lines = shallowRef<LogLine[]>([]);
    const paused = ref(false);
    const pending = shallowRef<LogLine[]>([]);
    const ended = ref(false);
    const endError = ref("");
    const failed = ref("");
    let nextId = 1;

    const url = computed(() => {
        const q = new URLSearchParams();
        const svcs = toValue(o.services);
        if (svcs.length) q.set("services", svcs.join(","));
        q.set("tail", String(toValue(o.tail)));
        const since = toValue(o.since);
        if (since) q.set("since", since);
        const ctx = toValue(o.context);
        if (ctx) q.set("context", ctx);
        const proto = location.protocol === "https:" ? "wss" : "ws";
        return `${proto}://${location.host}/ws/v1/projects/${toValue(o.projectId)}/logs?${q}`;
    });

    function publish(batch: LogLine[]) {
        if (!batch.length) return;
        const next = lines.value.concat(batch);
        lines.value =
            next.length > MAX_LINES
                ? next.slice(next.length - MAX_LINES)
                : next;
    }

    function onFrame(text: string) {
        let data: unknown;
        try {
            data = JSON.parse(text);
        } catch {
            return;
        }
        if (Array.isArray(data)) {
            const batch = (data as RawLine[]).map((r) =>
                toLogLine(r, nextId++),
            );
            if (paused.value) {
                const merged = pending.value.concat(batch);
                pending.value =
                    merged.length > MAX_LINES
                        ? merged.slice(merged.length - MAX_LINES)
                        : merged;
            } else {
                publish(batch);
            }
        } else if (
            data &&
            typeof data === "object" &&
            (data as { type?: string }).type === "end"
        ) {
            ended.value = true;
            endError.value = String((data as { error?: string }).error ?? "");
        }
    }

    const ws = useWebSocket(url, {
        immediate: true,
        autoReconnect: { retries: 5, delay: 1500 },
        onMessage: (_w, e) => onFrame(String(e.data)),
        onError: () => (failed.value = "connection error"),
        onConnected: () => (failed.value = ""),
    });

    function reset() {
        lines.value = [];
        pending.value = [];
        ended.value = false;
        endError.value = "";
        failed.value = "";
    }

    // every option change = a new stream; `url` changes only when an option does
    watch(url, reset);

    function pause() {
        paused.value = true;
    }
    function resume() {
        paused.value = false;
        if (pending.value.length) {
            publish(pending.value);
            pending.value = [];
        }
    }
    function clear() {
        lines.value = [];
        pending.value = [];
    }
    function restart() {
        reset();
        ws.close();
        ws.open();
    }

    return {
        lines,
        paused,
        buffered: computed(() => pending.value.length),
        ended,
        endError,
        failed,
        status: ws.status,
        live: computed(() => ws.status.value === "OPEN" && !ended.value),
        pause,
        resume,
        clear,
        restart,
        close: ws.close,
        url,
    };
}
