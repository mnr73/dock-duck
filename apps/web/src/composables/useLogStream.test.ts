import { nextTick, ref } from "vue";

import { beforeEach, describe, expect, it, vi } from "vitest";

import { MAX_LINES, useLogStream } from "./useLogStream";

// capture the options useWebSocket is created with so frames can be pushed by hand
let captured: { onMessage: (w: unknown, e: { data: string }) => void } | null =
    null;
vi.mock("@vueuse/core", () => ({
    useWebSocket: (
        _url: unknown,
        opts: { onMessage: (w: unknown, e: { data: string }) => void },
    ) => {
        captured = opts;
        return { status: ref("OPEN"), close: vi.fn(), open: vi.fn() };
    },
}));

const frame = (n: number, start = 0) =>
    JSON.stringify(
        Array.from({ length: n }, (_, i) => ({
            s: "app",
            t: "2026-11-21T08:00:00Z",
            m: `line ${start + i}`,
        })),
    );
const push = (data: string) => captured!.onMessage(null, { data });

function make() {
    const services = ref<string[]>([]);
    const s = useLogStream({ projectId: "p1", services, tail: 200, since: "" });
    return { s, services };
}

describe("useLogStream", () => {
    beforeEach(() => {
        captured = null;
        vi.stubGlobal("location", {
            protocol: "http:",
            host: "localhost:5173",
        });
    });

    it("builds the url from options", () => {
        const { s, services } = make();
        expect(s.url.value).toContain("/ws/v1/projects/p1/logs?tail=200");
        services.value = ["a", "b"];
        expect(decodeURIComponent(s.url.value)).toContain("services=a,b");
    });

    it("collects lines, classifies severity and ignores junk frames", () => {
        const { s } = make();
        push(frame(2));
        push("not json");
        push(JSON.stringify([{ s: "app", m: "FATAL boom" }]));
        expect(s.lines.value.map((l) => l.m)).toEqual([
            "line 0",
            "line 1",
            "FATAL boom",
        ]);
        expect(s.lines.value[2].level).toBe("error");
        expect(new Set(s.lines.value.map((l) => l.id)).size).toBe(3);
    });

    it("pause buffers new lines and resume flushes them in order", () => {
        const { s } = make();
        push(frame(2));
        s.pause();
        push(frame(3, 2));
        expect(s.lines.value).toHaveLength(2);
        expect(s.buffered.value).toBe(3);
        s.resume();
        expect(s.lines.value.map((l) => l.m)).toEqual([
            "line 0",
            "line 1",
            "line 2",
            "line 3",
            "line 4",
        ]);
        expect(s.buffered.value).toBe(0);
    });

    it("keeps at most MAX_LINES (drops the oldest)", () => {
        const { s } = make();
        push(frame(MAX_LINES - 1));
        push(frame(5, 100000));
        expect(s.lines.value).toHaveLength(MAX_LINES);
        expect(s.lines.value.at(-1)!.m).toBe("line 100004");
        expect(s.lines.value[0].m).toBe("line 4"); // first 4 dropped
    });

    it("clear empties the view but the stream keeps going", () => {
        const { s } = make();
        push(frame(3));
        s.clear();
        expect(s.lines.value).toHaveLength(0);
        push(frame(1, 9));
        expect(s.lines.value).toHaveLength(1);
    });

    it("end frame marks the stream ended with the error", () => {
        const { s } = make();
        push(JSON.stringify({ type: "end", error: "no such service" }));
        expect(s.ended.value).toBe(true);
        expect(s.endError.value).toBe("no such service");
        expect(s.live.value).toBe(false);
    });

    it("changing an option starts from an empty buffer", async () => {
        const { s, services } = make();
        push(frame(3));
        services.value = ["db"];
        await nextTick();
        expect(s.lines.value).toHaveLength(0);
    });
});
