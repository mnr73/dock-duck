import {
    type Ref,
    onBeforeUnmount,
    onMounted,
    ref,
    shallowRef,
    watch,
} from "vue";

import { useDebounceFn, useResizeObserver } from "@vueuse/core";
import { FitAddon } from "@xterm/addon-fit";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Terminal } from "@xterm/xterm";

import { terminalSocketUrl, terminalsApi } from "@/api/terminals";

export type TerminalState =
    "connecting" | "live" | "exited" | "taken" | "lost" | "gone";

export interface TerminalTheme {
    background: string;
    foreground: string;
    cursor: string;
    selectionBackground: string;
}

const encoder = new TextEncoder();

/**
 * xterm.js attached to a server-side terminal session over a binary WebSocket.
 *
 * Why it behaves like a real terminal (the things Docker Desktop's embedded one
 * gets wrong):
 *  - input is sent as raw bytes, so arrow keys, history, Home/End, Ctrl+keys,
 *    Alt+keys and bracketed paste reach the shell untouched;
 *  - the server runs the program under a real PTY with TERM=xterm-256color;
 *  - the size is sent on every resize (SIGWINCH), vim/htop/less redraw correctly;
 *  - on re-attach the old output is replayed, and terminal *queries* inside the
 *    replay (cursor position, device attributes) are not answered a second time.
 */
export function useTerminal(
    host: Ref<HTMLElement | null>,
    sessionId: () => string,
    opts: {
        fontSize: Ref<number>;
        fontFamily: string;
        theme: () => TerminalTheme;
    },
) {
    const state = ref<TerminalState>("connecting");
    const exitCode = ref<number | null>(null);
    const term = shallowRef<Terminal>();
    const title = ref("");

    let ws: WebSocket | undefined;
    let fit: FitAddon | undefined;
    let replayLeft = 0; // bytes of the replay frame still to come
    let suppressAnswers = false;
    let closedByUs = false;
    let retries = 0;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    function sendResize() {
        const t = term.value;
        if (!t || ws?.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "resize", cols: t.cols, rows: t.rows }));
    }

    const refit = useDebounceFn(() => {
        if (!host.value?.offsetWidth || !host.value.offsetHeight) return; // hidden tab: fitting would give 0x0
        try {
            fit?.fit();
        } catch {
            /* the container is not laid out yet */
        }
        sendResize();
    }, 60);

    function connect() {
        const t = term.value;
        if (!t) return;
        state.value = "connecting";
        ws = new WebSocket(terminalSocketUrl(sessionId()));
        ws.binaryType = "arraybuffer";
        const mine = ws;

        ws.onopen = () => {
            retries = 0;
            sendResize(); // the server needs our size before it draws the prompt
        };

        ws.onmessage = (ev) => {
            if (typeof ev.data === "string") {
                const m = JSON.parse(ev.data) as {
                    type: string;
                    code?: number;
                    replay?: number;
                };
                if (m.type === "attached") {
                    t.reset(); // a re-attach replays everything: start from a clean screen
                    replayLeft = m.replay ?? 0;
                    suppressAnswers = replayLeft > 0;
                    state.value = "live";
                    if (!suppressAnswers) t.focus();
                } else if (m.type === "exit") {
                    exitCode.value = m.code ?? null;
                    state.value = "exited";
                } else if (m.type === "taken") {
                    state.value = "taken";
                } else if (m.type === "gone") {
                    state.value = "gone"; // DockDuck is stopping: its shells end with it
                }
                return;
            }
            const bytes = new Uint8Array(ev.data as ArrayBuffer);
            if (replayLeft > 0) {
                replayLeft -= bytes.length;
                t.write(bytes, () => {
                    if (replayLeft <= 0) {
                        suppressAnswers = false;
                        t.scrollToBottom();
                        t.focus();
                    }
                });
                return;
            }
            t.write(bytes);
        };

        ws.onclose = () => {
            if (mine !== ws) return;
            if (
                state.value === "exited" ||
                state.value === "taken" ||
                state.value === "gone" ||
                closedByUs
            )
                return;
            // the browser lost the server (restart, sleep, network): a shell on a server
            // that is still running keeps going, so re-attaching gets the screen back
            state.value = "lost";
            scheduleRetry();
        };
    }

    function scheduleRetry() {
        if (retries >= 6) return;
        const wait = Math.min(1000 * 2 ** retries, 8000);
        retries++;
        retryTimer = setTimeout(async () => {
            if (closedByUs) return;
            try {
                // the server answers but does not know the session: it was restarted, the shell is gone
                const all = await terminalsApi.list();
                if (!all.some((x) => x.id === sessionId())) {
                    state.value = "gone";
                    return;
                }
            } catch {
                scheduleRetry(); // still unreachable
                return;
            }
            connect();
        }, wait);
    }

    onMounted(() => {
        if (!host.value) return;
        const t = new Terminal({
            fontFamily: opts.fontFamily,
            fontSize: opts.fontSize.value,
            lineHeight: 1.15,
            cursorBlink: true,
            scrollback: 10000,
            allowProposedApi: true,
            macOptionIsMeta: true, // Option acts as Alt on macOS: Alt+b / Alt+f / Alt+. in readline
            rightClickSelectsWord: true,
            theme: opts.theme(),
        });
        fit = new FitAddon();
        t.loadAddon(fit);
        t.loadAddon(new Unicode11Addon());
        t.unicode.activeVersion = "11";
        t.loadAddon(new WebLinksAddon());
        t.open(host.value);
        term.value = t;

        // keyboard -> raw bytes. xterm.js already turns Up/Down/Left/Right/Home/End/F-keys
        // into the escape sequences the program expects (application cursor mode included).
        t.onData((d) => {
            if (suppressAnswers || ws?.readyState !== WebSocket.OPEN) return;
            ws.send(encoder.encode(d));
        });
        t.onBinary((d) => {
            if (suppressAnswers || ws?.readyState !== WebSocket.OPEN) return;
            const b = new Uint8Array(d.length);
            for (let i = 0; i < d.length; i++) b[i] = d.charCodeAt(i) & 0xff;
            ws.send(b);
        });
        t.onTitleChange((s) => (title.value = s));
        t.onResize(() => sendResize());

        // copy/paste the way terminals do it on Linux/Windows: Ctrl+Shift+C / V,
        // while plain Ctrl+C stays an interrupt (unless there is a selection and
        // the user is on macOS, where Cmd+C is handled by the browser)
        t.attachCustomKeyEventHandler((e) => {
            if (e.type !== "keydown") return true;
            if (e.ctrlKey && e.shiftKey && e.code === "KeyC") {
                const sel = t.getSelection();
                if (sel) void navigator.clipboard?.writeText(sel);
                return false;
            }
            if (e.ctrlKey && e.shiftKey && e.code === "KeyV") {
                void navigator.clipboard
                    ?.readText()
                    .then((s) => s && t.paste(s));
                return false;
            }
            return true;
        });

        useResizeObserver(host, () => refit());
        void document.fonts?.ready.then(() => refit()); // glyph metrics change when the font loads
        refit();
        connect();
    });

    // a finished session must not look alive: no blinking, a hollow cursor
    watch(state, (st) => {
        const t = term.value;
        if (!t) return;
        const over = st === "exited" || st === "taken" || st === "gone";
        t.options.cursorBlink = !over;
        t.options.cursorStyle = over ? "underline" : "block";
        t.options.cursorInactiveStyle = over ? "none" : "outline";
    });

    watch(opts.fontSize, (n) => {
        if (!term.value) return;
        term.value.options.fontSize = n;
        refit();
    });
    watch(
        () => opts.theme(),
        (th) => {
            if (term.value) term.value.options.theme = th;
        },
    );

    onBeforeUnmount(() => {
        closedByUs = true;
        clearTimeout(retryTimer);
        ws?.close(); // the shell stays alive on the server; the tab list re-attaches
        term.value?.dispose();
    });

    function reconnect() {
        retries = 0;
        clearTimeout(retryTimer);
        ws?.close();
        state.value = "lost";
        scheduleRetry();
    }

    return {
        state,
        exitCode,
        title,
        term,
        reconnect,
        focus: () => term.value?.focus(),
        refit: () => refit(),
        paste: (s: string) => term.value?.paste(s),
        clear: () => term.value?.clear(),
        selection: () => term.value?.getSelection() ?? "",
    };
}
