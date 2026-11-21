import { describe, expect, it } from "vitest";

import {
    buildMatcher,
    detectLevel,
    formatLine,
    formatTime,
    highlight,
    matches,
    passesLevel,
    serviceColor,
    stripAnsi,
    toLogLine,
} from "./logs";

describe("ansi + level", () => {
    it("strips colour codes", () => {
        expect(
            stripAnsi("\x1b[31mred\x1b[0m plain \x1b[1;32mgreen\x1b[0m"),
        ).toBe("red plain green");
        expect(stripAnsi("no codes")).toBe("no codes");
    });
    it("detects severity", () => {
        expect(detectLevel("PHP Fatal error: x")).toBe("error");
        expect(detectLevel("[ERROR] boom")).toBe("error");
        expect(detectLevel("WARN disk almost full")).toBe("warn");
        expect(detectLevel("Deprecated: foo()")).toBe("warn");
        expect(detectLevel("DEBUG cache miss")).toBe("debug");
        expect(detectLevel("GET /api 200")).toBe("info");
        expect(detectLevel("terror strikes")).toBe("info"); // word boundary
        // formats from real containers
        expect(
            detectLevel(
                "[Sat Nov 21 05:30:17.117] [core:error] [pid 1] AH00126: Invalid URI",
            ),
        ).toBe("error");
        expect(
            detectLevel(
                "[Sat Nov 21 05:30:17.117] [mpm_prefork:notice] [pid 1] AH00163: configured",
            ),
        ).toBe("info");
        expect(
            detectLevel(
                "AH00558: apache2: Could not reliably determine the server's fully qualified domain name",
            ),
        ).toBe("warn");
        expect(detectLevel("[core:warn] [pid 1] something odd")).toBe("warn");
        expect(detectLevel('{"level":"error","msg":"db down"}')).toBe("error");
        expect(detectLevel("time=2026 level=warn msg=slow")).toBe("warn");
        expect(detectLevel("level=info msg=ok")).toBe("info");
        expect(detectLevel("[core:debug] [pid 1] cache miss")).toBe("debug");
        expect(
            detectLevel('172.18.0.1 - - "GET /index.php HTTP/1.1" 200 2450'),
        ).toBe("info");
        expect(detectLevel("connection refused by upstream")).toBe("warn");
        expect(detectLevel("panic: runtime error: index out of range")).toBe(
            "error",
        );
        expect(detectLevel("information about errors-free run")).toBe("info");
    });
    it("level filter is a minimum severity", () => {
        expect(passesLevel("info", "all")).toBe(true);
        expect(passesLevel("info", "warn")).toBe(false);
        expect(passesLevel("warn", "warn")).toBe(true);
        expect(passesLevel("error", "warn")).toBe(true);
        expect(passesLevel("warn", "error")).toBe(false);
    });
    it("builds lines", () => {
        const l = toLogLine(
            {
                s: "app",
                t: "2026-11-21T08:00:00Z",
                m: "\x1b[31mERROR x\x1b[0m",
            },
            7,
        );
        expect(l).toMatchObject({
            id: 7,
            s: "app",
            m: "ERROR x",
            level: "error",
        });
    });
});

describe("search", () => {
    it("escapes plain text, honours regex mode, reports bad patterns", () => {
        expect(matches("a.b", buildMatcher("a.b", false))).toBe(true);
        expect(matches("axb", buildMatcher("a.b", false))).toBe(false);
        expect(matches("axb", buildMatcher("a.b", true))).toBe(true);
        expect(matches("anything", buildMatcher("", false))).toBe(true);
        const bad = buildMatcher("(", true);
        expect(bad.re).toBeNull();
        expect(bad.error).not.toBe("");
    });
    it("is case-insensitive and repeatable (no lastIndex leaks)", () => {
        const m = buildMatcher("error", false);
        expect([
            matches("An ERROR", m),
            matches("An ERROR", m),
            matches("an error", m),
        ]).toEqual([true, true, true]);
    });
    it("highlights all hits and ignores empty matches", () => {
        expect(highlight("foo bar foo", buildMatcher("foo", false))).toEqual([
            { text: "foo", hit: true },
            { text: " bar ", hit: false },
            { text: "foo", hit: true },
        ]);
        expect(highlight("abc", buildMatcher("x*", true))).toEqual([
            { text: "abc", hit: false },
        ]);
        expect(highlight("abc", buildMatcher("", false))).toEqual([
            { text: "abc", hit: false },
        ]);
    });
});

describe("format", () => {
    it("time and colours", () => {
        expect(formatTime("")).toBe("");
        expect(formatTime("garbage")).toBe("");
        expect(formatTime("2026-11-21T08:03:50.772435037Z")).toMatch(
            /^\d{2}:\d{2}:\d{2}\.772$/,
        );
        const all = ["a", "b"];
        expect(serviceColor("a", all)).not.toBe(serviceColor("b", all));
        expect(serviceColor("zzz", all)).toMatch(/^#/);
    });
    it("plain-text line", () => {
        const l = toLogLine({ s: "app", t: "T", m: "hi" }, 1);
        expect(formatLine(l, true)).toBe("app | T hi");
        expect(formatLine(l, false, 6)).toBe("app    | hi");
    });
});
