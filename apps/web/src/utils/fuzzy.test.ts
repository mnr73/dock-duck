import { describe, expect, it } from "vitest";

import { bestScore, fuzzyScore } from "./fuzzy";

describe("fuzzyScore", () => {
    it("matches everything for an empty query", () => {
        expect(fuzzyScore("", "anything")).toBe(0);
        expect(fuzzyScore("   ", "anything")).toBe(0);
    });

    it("needs every character, in order", () => {
        expect(fuzzyScore("xyz", "settings")).toBeNull();
        expect(fuzzyScore("stg", "settings")).not.toBeNull();
        expect(fuzzyScore("tes", "settings")).toBeNull(); // no "e" after the first "t"
        expect(fuzzyScore("gts", "settings")).toBeNull(); // g comes after t,s
    });

    it("ranks exact, prefix, substring, subsequence in that order", () => {
        const exact = fuzzyScore("ports", "ports")!;
        const prefix = fuzzyScore("por", "ports")!;
        const sub = fuzzyScore("ort", "ports")!;
        const seq = fuzzyScore("pts", "ports")!;
        expect(exact).toBeLessThan(prefix);
        expect(prefix).toBeLessThan(sub);
        expect(sub).toBeLessThan(seq);
        // word initials sit between a prefix and a mid-word substring
        const initials = fuzzyScore("rp", "Rescan projects")!;
        expect(prefix).toBeLessThan(initials);
        expect(initials).toBeLessThan(fuzzyScore("rp", "sharp")!);
    });

    it("prefers word starts: 'rp' finds 'Rescan projects' over 'sharp tool'", () => {
        expect(fuzzyScore("rp", "Rescan projects")!).toBeLessThan(
            fuzzyScore("rp", "sharp tool")!,
        );
    });

    it("is case-insensitive and treats the query as plain text", () => {
        expect(fuzzyScore("ERP", "erp-back")).not.toBeNull();
        expect(fuzzyScore(".*", "erp-back")).toBeNull(); // no regex semantics
    });

    it("a shorter name beats a longer one on the same prefix", () => {
        expect(fuzzyScore("log", "logs")!).toBeLessThan(
            fuzzyScore("log", "logging-service")!,
        );
    });
});

describe("bestScore", () => {
    it("takes the best field and ignores fields that do not match", () => {
        expect(bestScore("back", ["zhoon", "/home/u/erp-back"])).not.toBeNull();
        expect(bestScore("zzz", ["a", "b"])).toBeNull();
        expect(bestScore("erp", ["x-erp", "erp"])).toBe(0);
    });
});
