import { describe, expect, it } from "vitest";

import {
    containerTone,
    projectLabel,
    projectTone,
    relativeTo,
    serviceTone,
    shortPath,
} from "./status";

describe("status tones", () => {
    it("maps project status", () => {
        expect(projectTone("running")).toBe("ok");
        expect(projectTone("partial")).toBe("warn");
        expect(projectTone("stopped")).toBe("off");
        expect(projectTone("unknown")).toBe("off");
    });
    it("maps service status", () => {
        expect(serviceTone("unhealthy")).toBe("err");
        expect(serviceTone("error")).toBe("err");
        expect(serviceTone("restarting")).toBe("warn");
        expect(serviceTone("exited")).toBe("off");
    });
    it("maps containers by exit code and health", () => {
        expect(containerTone({ state: "exited", exit_code: 3 })).toBe("err");
        expect(containerTone({ state: "exited", exit_code: 0 })).toBe("off");
        expect(
            containerTone({
                state: "running",
                health: "unhealthy",
                exit_code: 0,
            }),
        ).toBe("err");
        expect(
            containerTone({
                state: "running",
                health: "starting",
                exit_code: 0,
            }),
        ).toBe("warn");
        expect(containerTone({ state: "running", exit_code: 0 })).toBe("ok");
    });
});

describe("labels and paths", () => {
    it("labels", () => {
        expect(projectLabel("partial", 5, 7)).toBe("Degraded · 5 of 7 up");
        expect(projectLabel("stopped", 0, 3)).toBe("Stopped");
    });
    it("shortens home", () => {
        expect(shortPath("/home/me/prj/x", "/home/me")).toBe("~/prj/x");
        expect(shortPath("/home/meow/x", "/home/me")).toBe("/home/meow/x");
        expect(shortPath("/srv/x", "")).toBe("/srv/x");
    });
    it("relative to root", () => {
        expect(relativeTo("/r/a/b", "/r")).toBe("a/b");
        expect(relativeTo("/x/a", "/r")).toBe("/x/a");
        expect(relativeTo("/x/a", undefined)).toBe("/x/a");
    });
});
