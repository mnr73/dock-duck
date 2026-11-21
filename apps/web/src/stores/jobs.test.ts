import { describe, expect, it } from "vitest";

import { lastUsefulLine, parseDone } from "./jobs";

describe("parseDone", () => {
    it("parses the server control frame whatever the key order", () => {
        expect(
            parseDone('{"error":"","status":"succeeded","type":"done"}'),
        ).toEqual({ status: "succeeded", error: "" });
        expect(
            parseDone('{"type":"done","status":"failed","error":"exit 1"}'),
        ).toEqual({ status: "failed", error: "exit 1" });
    });
    it("leaves ordinary log lines alone", () => {
        expect(parseDone(" Container x Started")).toBeNull();
        expect(
            parseDone('{"level":"info","msg":"the job is done"}'),
        ).toBeNull();
        expect(parseDone('{"done": true}')).toBeNull();
        expect(parseDone("{ not json done")).toBeNull();
    });
});

describe("lastUsefulLine", () => {
    it("prefers the last error-looking line", () => {
        expect(
            lastUsefulLine(["a", "Error: port is already allocated", "b"]),
        ).toBe("Error: port is already allocated");
        expect(lastUsefulLine(["a", "", "last"])).toBe("last");
        expect(lastUsefulLine([])).toBe("");
    });
});
