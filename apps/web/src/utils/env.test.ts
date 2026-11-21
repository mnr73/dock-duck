import { describe, expect, it } from "vitest";

import type { EnvLine } from "@/api/types";

import {
    MASK,
    countEnvChanges,
    countLineChanges,
    newRow,
    parseEnv,
    rowsToText,
    toRows,
    validateRows,
} from "./env";

const lines: EnvLine[] = [
    { kind: "comment", raw: "# app" },
    { kind: "var", key: "APP_NAME", value: "Shiva", quote: "" },
    { kind: "var", key: "DEBUG", value: "true", export: true },
    { kind: "blank" },
    {
        kind: "var",
        key: "DB_PASSWORD",
        value: "s3 cret # x",
        quote: '"',
        secret: true,
    },
    { kind: "invalid", raw: 'MULTI="line one' },
];

describe("rowsToText", () => {
    it("round-trips comments, blanks, export, quotes and unparsed lines", () => {
        expect(rowsToText(toRows(lines))).toBe(
            '# app\nAPP_NAME=Shiva\nexport DEBUG=true\n\nDB_PASSWORD="s3 cret # x"\nMULTI="line one\n',
        );
    });
    it("quotes only when needed to read the value back", () => {
        const q = (v: string) => rowsToText([newRow("K", v)]);
        expect(q("plain")).toBe("K=plain\n");
        expect(q("with space")).toBe('K="with space"\n');
        expect(q("a#b")).toBe('K="a#b"\n');
        expect(q(" lead")).toBe('K=" lead"\n');
        expect(q('say "hi" now')).toBe("K='say \"hi\" now'\n");
        expect(q("")).toBe("K=\n");
        expect(q("http://x/?a=1&b=2")).toBe("K=http://x/?a=1&b=2\n");
    });
    it("keeps the original quote style", () => {
        const rows = toRows([
            { kind: "var", key: "URL", value: "http://x", quote: "'" },
        ]);
        expect(rowsToText(rows)).toBe("URL='http://x'\n");
    });
    it("empty table is an empty file", () => {
        expect(rowsToText([])).toBe("");
    });
});

describe("validateRows", () => {
    it("flags empty, malformed and duplicate names", () => {
        const rows = [
            newRow("OK", "1"),
            newRow("", "x"),
            newRow("bad name", "x"),
            newRow("1ST", "x"),
            newRow("OK", "2"),
        ];
        const msgs = validateRows(rows).map((p) => p.message);
        expect(msgs).toEqual([
            "Name is empty",
            expect.stringContaining("no spaces"),
            expect.stringContaining("no spaces"),
            "Duplicate of an earlier OK",
        ]);
    });
    it("accepts the usual names", () => {
        expect(
            validateRows([newRow("APP_NAME"), newRow("a.b-c"), newRow("_X")]),
        ).toEqual([]);
    });
    it("ignores comments", () => {
        expect(validateRows(toRows([{ kind: "comment", raw: "# x" }]))).toEqual(
            [],
        );
    });
});

describe("change counters", () => {
    it("counts variable changes, not text noise", () => {
        expect(countEnvChanges("A=1\nB=2\n", "A=1\nB=2\n")).toBe(0);
        expect(countEnvChanges("A=1\nB=2\n", "# new comment\nA=1\nB=2\n")).toBe(
            0,
        );
        expect(countEnvChanges("A=1\nB=2\n", "A=1\nB=3\nC=4\n")).toBe(2);
        expect(countEnvChanges("A=1\nB=2\n", "A=1\n")).toBe(1);
    });
    it("counts changed lines", () => {
        expect(countLineChanges("a\nb\nc", "a\nb\nc")).toBe(0);
        expect(countLineChanges("a\nb\nc", "a\nX\nc")).toBe(1);
        expect(countLineChanges("a\nb\nc", "a\nb\nc\nd\ne")).toBe(2);
        expect(countLineChanges("a\nb\nc", "a")).toBe(2);
    });
});

describe("secrets", () => {
    it("newRow detects secret names like the server", () => {
        expect(newRow("API_TOKEN").secret).toBe(true);
        expect(newRow("APP_NAME").secret).toBe(false);
    });
});

describe("parseEnv (mirrors the server parser)", () => {
    const text =
        '# app\nAPP_NAME=Shiva\nexport DEBUG=true\n\nDB_PASSWORD="s3 cret # x"\nTRAIL=v # c\nbad line\nMULTI="line one\nrest"\n';
    it("classifies lines like the server", () => {
        expect(parseEnv(text).map((r) => r.kind)).toEqual([
            "comment",
            "var",
            "var",
            "blank",
            "var",
            "var",
            "invalid",
            "invalid",
            "invalid",
        ]);
        const by = Object.fromEntries(
            parseEnv(text)
                .filter((r) => r.kind === "var")
                .map((r) => [r.key, r]),
        );
        expect(by.DB_PASSWORD).toMatchObject({
            value: "s3 cret # x",
            quote: '"',
            secret: true,
        });
        expect(by.TRAIL.value).toBe("v");
        expect(by.DEBUG.export).toBe(true);
    });
    it("text -> rows -> text is lossless for normal files", () => {
        const ok =
            '# app\nAPP_NAME=Shiva\nexport DEBUG=true\n\nDB_PASSWORD="s3 cret # x"\nURL=\'http://x\'\nMULTI="line one\nrest"\n';
        expect(rowsToText(parseEnv(ok))).toBe(ok);
    });
    it("recognises the server's mask and never turns it into a value", () => {
        const rows = parseEnv(`APP_NAME=x\nDB_PASSWORD=${MASK}\n`);
        expect(rows[1]).toMatchObject({ masked: true, value: "" });
    });
    it("empty text is no rows", () => {
        expect(parseEnv("")).toEqual([]);
        expect(parseEnv("\n")).toEqual([]);
    });
});

describe("masked secrets never become empty values", () => {
    it("serialize as the mask, so Raw shows it and Save is blocked", () => {
        const rows = toRows([
            { kind: "var", key: "APP_NAME", value: "x" },
            {
                kind: "var",
                key: "DB_PASSWORD",
                value: "",
                secret: true,
                masked: true,
                set: true,
            },
        ]);
        expect(rowsToText(rows)).toBe(`APP_NAME=x\nDB_PASSWORD=${MASK}\n`);
    });
    it("survive Table -> Raw -> Table", () => {
        const rows = toRows([
            {
                kind: "var",
                key: "API_TOKEN",
                value: "",
                secret: true,
                masked: true,
            },
        ]);
        const back = parseEnv(rowsToText(rows));
        expect(back[0]).toMatchObject({
            key: "API_TOKEN",
            masked: true,
            value: "",
        });
        expect(rowsToText(back)).toBe(rowsToText(rows));
    });
    it("a revealed secret that the user cleared is a real empty value", () => {
        const rows = toRows([
            { kind: "var", key: "API_TOKEN", value: "abc", secret: true },
        ]);
        rows[0].value = "";
        expect(rowsToText(rows)).toBe("API_TOKEN=\n");
    });
});
