import { describe, expect, it } from "vitest";

import { bindLabel, formatBytes, formatCpu, isAnyIP, portUrl } from "./format";

describe("format", () => {
    it("bind labels", () => {
        expect(bindLabel(["0.0.0.0", "::"])).toBe("all interfaces");
        expect(bindLabel(["127.0.0.1"])).toBe("localhost only");
        expect(bindLabel(["192.168.1.5"])).toBe("192.168.1.5");
        expect(isAnyIP("")).toBe(true);
    });
    it("port url", () => {
        expect(portUrl(8080, 80)).toBe("http://localhost:8080");
        expect(portUrl(8443, 443)).toBe("https://localhost:8443");
    });
    it("bytes and cpu", () => {
        expect(formatBytes(0)).toBe("0 B");
        expect(formatBytes(487273267)).toBe("465 MiB");
        expect(formatBytes(11188305)).toBe("10.7 MiB");
        expect(formatCpu(0.4)).toBe("0.4%");
        expect(formatCpu(13.64)).toBe("14%");
    });
});
