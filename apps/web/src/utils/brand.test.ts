import { describe, expect, it } from "vitest";

import { brandSvg, tint } from "./brand";

describe("brand", () => {
    it("keeps the supplied artwork when no accent is chosen", () => {
        const s = brandSvg(null);
        expect(s).toContain("#F1B74C");
        expect(s).toContain("#F8D996");
    });

    it("recolours both fills for an accent and drops the old ones", () => {
        const s = brandSvg("#45c46b");
        expect(s).toContain("#45c46b");
        expect(s).toContain(tint("#45c46b", 0.42));
        expect(s).not.toContain("#F1B74C");
        expect(s).not.toContain("#F8D996");
    });

    it("derives the original tint from the original colour (within a few levels)", () => {
        const [r, g, b] = [0, 2, 4].map((i) =>
            Number.parseInt(tint("#F1B74C", 0.42).slice(1 + i, 3 + i), 16),
        );
        expect(Math.abs(r - 0xf8)).toBeLessThanOrEqual(4);
        expect(Math.abs(g - 0xd9)).toBeLessThanOrEqual(6);
        expect(Math.abs(b - 0x96)).toBeLessThanOrEqual(8);
    });
});
