import type { EnvLine } from "@/api/types";

/** A row of the table editor. `id` is stable while editing; `original` is what was loaded. */
export interface EnvRow {
    id: number;
    kind: EnvLine["kind"];
    key: string;
    value: string;
    quote: "" | '"' | "'";
    export: boolean;
    raw: string;
    secret: boolean;
    /** the real value is not in the browser (masked on the server); `value` is empty until revealed */
    masked: boolean;
    /** a variable the user added or edited in this session */
    touched: boolean;
}

/** What the server puts in place of a secret value in masked text (api/files.go). */
export const MASK = "••••••••";

let nextRowId = 1;

export function toRows(lines: EnvLine[]): EnvRow[] {
    return lines.map((l) => ({
        id: nextRowId++,
        kind: l.kind,
        key: l.key ?? "",
        value: l.value ?? "",
        quote: l.quote ?? "",
        export: !!l.export,
        raw: l.raw ?? "",
        secret: !!l.secret,
        masked: !!l.masked,
        touched: false,
    }));
}

export function newRow(key = "", value = ""): EnvRow {
    return {
        id: nextRowId++,
        kind: "var",
        key,
        value,
        quote: "",
        export: false,
        raw: "",
        secret: isSecretKey(key),
        masked: false,
        touched: true,
    };
}

const SECRET_WORDS = [
    "PASSWORD",
    "PASSWD",
    "SECRET",
    "TOKEN",
    "KEY",
    "CREDENTIAL",
    "PRIVATE",
    "AUTH",
    "DSN",
    "SALT",
    "PASS",
];

/** Same heuristic as the server (files.IsSecretKey). */
export function isSecretKey(k: string): boolean {
    const u = k.toUpperCase();
    return SECRET_WORDS.some((w) => u.includes(w));
}

export const KEY_RE = /^[A-Za-z_][A-Za-z0-9_.-]*$/;

/** Quote a value only when needed to read it back unchanged. */
function formatValue(r: EnvRow): string {
    const v = r.value;
    const needs = /[\s#"'\\]/.test(v) && !(r.quote && !v.includes(r.quote));
    if (r.quote) {
        if (v.includes(r.quote)) return v.includes('"') ? `'${v}'` : `"${v}"`;
        return `${r.quote}${v}${r.quote}`;
    }
    if (needs || v !== v.trim()) return v.includes('"') ? `'${v}'` : `"${v}"`;
    return v;
}

/** Rows back to .env text; comments/blank/unparsed lines are kept as they were. */
export function rowsToText(rows: EnvRow[]): string {
    const out = rows.map((r) => {
        switch (r.kind) {
            case "blank":
                return "";
            case "comment":
            case "invalid":
                return r.raw;
            default:
                // a hidden secret must never serialize as an empty value: it would
                // look like "the user cleared it" and erase the real one on save
                return `${r.export ? "export " : ""}${r.key}=${r.masked ? MASK : formatValue(r)}`;
        }
    });
    return out.length ? out.join("\n") + "\n" : "";
}

export interface EnvProblem {
    row: number;
    message: string;
}

/** Problems that make the table unsaveable: bad or duplicate keys. */
export function validateRows(rows: EnvRow[]): EnvProblem[] {
    const out: EnvProblem[] = [];
    const seen = new Map<string, number>();
    for (const r of rows) {
        if (r.kind !== "var") continue;
        if (!r.key) {
            out.push({ row: r.id, message: "Name is empty" });
            continue;
        }
        if (!KEY_RE.test(r.key))
            out.push({
                row: r.id,
                message:
                    "Use letters, digits and _ . - (no spaces; not starting with a digit)",
            });
        if (seen.has(r.key))
            out.push({
                row: r.id,
                message: `Duplicate of an earlier ${r.key}`,
            });
        seen.set(r.key, r.id);
    }
    return out;
}

/** Number of variables that differ between two texts (added + removed + changed values). */
export function countEnvChanges(before: string, after: string): number {
    const a = varsOf(before);
    const b = varsOf(after);
    let n = 0;
    for (const [k, v] of b) if (!a.has(k) || a.get(k) !== v) n++;
    for (const k of a.keys()) if (!b.has(k)) n++;
    return n;
}

function varsOf(text: string): Map<string, string> {
    const m = new Map<string, string>();
    for (const raw of text.split(/\r?\n/)) {
        const t = raw.trim();
        if (!t || t.startsWith("#")) continue;
        const body = t.startsWith("export ") ? t.slice(7).trim() : t;
        const i = body.indexOf("=");
        if (i > 0) m.set(body.slice(0, i).trim(), body.slice(i + 1).trim());
    }
    return m;
}

/** Lines that differ between two texts (for the "N lines changed" badge). */
export function countLineChanges(before: string, after: string): number {
    if (before === after) return 0;
    const a = before.split("\n");
    const b = after.split("\n");
    // longest common prefix/suffix, the rest counts as changed
    let s = 0;
    while (s < a.length && s < b.length && a[s] === b[s]) s++;
    let ea = a.length - 1;
    let eb = b.length - 1;
    while (ea >= s && eb >= s && a[ea] === b[eb]) {
        ea--;
        eb--;
    }
    return Math.max(ea - s + 1, eb - s + 1, 0);
}

/** Client mirror of the server's files.ParseEnv, so Table and Raw stay in sync without a request. */
export function parseEnv(text: string): EnvRow[] {
    const src = text.replace(/\r\n/g, "\n").replace(/\n$/, "");
    if (src === "") return [];
    return src.split("\n").map((raw): EnvRow => {
        const base = {
            ...newRow(),
            touched: false,
            raw,
            kind: "invalid" as EnvLine["kind"],
        };
        const t = raw.trim();
        if (t === "") return { ...base, kind: "blank", raw: "" };
        if (t.startsWith("#")) return { ...base, kind: "comment" };
        let body = t;
        let exported = false;
        if (body.startsWith("export ")) {
            exported = true;
            body = body.slice(7).trim();
        }
        const i = body.indexOf("=");
        const key = i > 0 ? body.slice(0, i).trim() : "";
        if (!key || !KEY_RE.test(key)) return base;
        let v = body.slice(i + 1).trim();
        let quote: EnvRow["quote"] = "";
        if (
            v.length >= 2 &&
            (v[0] === '"' || v[0] === "'") &&
            v.endsWith(v[0])
        ) {
            quote = v[0] as EnvRow["quote"];
            v = v.slice(1, -1);
        } else if (v[0] === '"' || v[0] === "'") {
            return base; // multi-line value: kept as an untouched raw line
        } else {
            const c = v.indexOf(" #");
            if (c >= 0) v = v.slice(0, c).trim();
        }
        const masked = v === MASK;
        return {
            ...base,
            kind: "var",
            key,
            value: masked ? "" : v,
            quote,
            export: exported,
            secret: isSecretKey(key),
            masked,
            raw: "",
        };
    });
}
