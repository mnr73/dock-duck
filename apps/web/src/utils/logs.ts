export type LogLevel = "error" | "warn" | "info" | "debug";

export interface LogLine {
    id: number;
    /** service name (or raw label when unknown) */
    s: string;
    /** RFC3339Nano timestamp from Docker, may be empty */
    t: string;
    /** message with ANSI escapes removed */
    m: string;
    level: LogLevel;
}

// eslint-disable-next-line no-control-regex
const ANSI = /\x1b\[[0-9;?]*[ -/]*[@-~]/g;

export function stripAnsi(s: string): string {
    return s.replace(ANSI, "");
}

// Order matters: the first group that matches wins (error > warn > debug).
// Formats seen in real logs: `[error]`, `ERROR:`, `level=warn`, `"level":"error"`,
// Apache `[core:error]` / `[mpm_prefork:notice]`, PHP `PHP Fatal error`, Go `panic:`,
// Python `Traceback`, syslog-ish `<err>`. Apache's AHxxxxx codes carry no level word,
// so `AH00558` style warnings are matched by their known "could not reliably" text only.
const ERROR_RE =
    /\b(error|fatal|critical|crit|alert|emerg|panic|exception|traceback|segfault)\b|\[[\w.]+:(error|crit|alert|emerg)\]|level[=":\s]+"?(error|fatal|critical)|\bE\//i;
const WARN_RE =
    /\b(warn|warning|deprecated)\b|\[[\w.]+:warn\]|level[=":\s]+"?warn|could not reliably determine|\b(failed|failure|denied|refused|timed? out|timeout)\b/i;
const DEBUG_RE =
    /\b(debug|trace|verbose)\b|\[[\w.]+:(debug|trace\d?)\]|level[=":\s]+"?(debug|trace)/i;

/** Heuristic severity of a log message. */
export function detectLevel(m: string): LogLevel {
    if (ERROR_RE.test(m)) return "error";
    if (WARN_RE.test(m)) return "warn";
    if (DEBUG_RE.test(m)) return "debug";
    return "info";
}

export interface RawLine {
    s: string;
    t?: string;
    m: string;
}

export function toLogLine(raw: RawLine, id: number): LogLine {
    const m = stripAnsi(raw.m);
    return { id, s: raw.s, t: raw.t ?? "", m, level: detectLevel(m) };
}

/** Level filter: "all" or the minimum severity to show. */
export type LevelFilter = "all" | "warn" | "error";

const RANK: Record<LogLevel, number> = { debug: 0, info: 1, warn: 2, error: 3 };

export function passesLevel(l: LogLevel, f: LevelFilter): boolean {
    if (f === "all") return true;
    return RANK[l] >= RANK[f === "warn" ? "warn" : "error"];
}

export interface Matcher {
    re: RegExp | null;
    error: string;
}

/** Builds a case-insensitive matcher; plain text is escaped unless `regex` is on. */
export function buildMatcher(query: string, regex: boolean): Matcher {
    if (!query) return { re: null, error: "" };
    try {
        const src = regex
            ? query
            : query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
        return { re: new RegExp(src, "gi"), error: "" };
    } catch (e) {
        return {
            re: null,
            error: e instanceof Error ? e.message : "invalid pattern",
        };
    }
}

export function matches(text: string, m: Matcher): boolean {
    if (!m.re) return true;
    m.re.lastIndex = 0;
    return m.re.test(text);
}

export interface Part {
    text: string;
    hit: boolean;
}

/** Splits text into plain and matching parts (zero-length matches are ignored). */
export function highlight(text: string, m: Matcher): Part[] {
    if (!m.re) return [{ text, hit: false }];
    const out: Part[] = [];
    let last = 0;
    m.re.lastIndex = 0;
    for (let r = m.re.exec(text); r; r = m.re.exec(text)) {
        if (r[0] === "") {
            m.re.lastIndex++;
            continue;
        }
        if (r.index > last)
            out.push({ text: text.slice(last, r.index), hit: false });
        out.push({ text: r[0], hit: true });
        last = r.index + r[0].length;
    }
    if (last < text.length) out.push({ text: text.slice(last), hit: false });
    return out.length ? out : [{ text, hit: false }];
}

/** HH:MM:SS.mmm in local time; "" for a missing/invalid timestamp. */
export function formatTime(t: string): string {
    if (!t) return "";
    const d = new Date(t);
    if (Number.isNaN(d.getTime())) return "";
    const p = (n: number, w = 2) => String(n).padStart(w, "0");
    return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}.${p(d.getMilliseconds(), 3)}`;
}

/** Colours for service labels on the always-dark log surface (from the mockup palette). */
const PALETTE = [
    "#7fb2ff",
    "#8fe0a6",
    "#f2c672",
    "#d9a6ff",
    "#7ed6c9",
    "#ff9a9a",
    "#9adbf5",
    "#ffd27a",
];

export function serviceColor(name: string, all: string[]): string {
    const i = all.indexOf(name);
    if (i >= 0) return PALETTE[i % PALETTE.length];
    let h = 0;
    for (const c of name) h = (h * 31 + c.charCodeAt(0)) | 0;
    return PALETTE[Math.abs(h) % PALETTE.length];
}

/** Plain-text form used for copy and export. */
export function formatLine(l: LogLine, withTime: boolean, width = 0): string {
    const svc = width ? l.s.padEnd(width) : l.s;
    const time = withTime && l.t ? `${l.t} ` : "";
    return `${svc} | ${time}${l.m}`;
}
