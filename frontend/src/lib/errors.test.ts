import { describe, expect, it } from "vitest";
import { friendlyError, isWorkspaceBoundaryError } from "./errors";

describe("error helpers", () => {
  it("detects workspace boundary errors from backend messages", () => {
    expect(isWorkspaceBoundaryError("path C:\\tmp\\x is outside the allowed workspace D:\\repo")).toBe(true);
    expect(isWorkspaceBoundaryError(new Error("cannot resolve target path: bad path"))).toBe(true);
  });

  it("maps workspace boundary errors to friendly copy", () => {
    expect(friendlyError("tool read_file blocked: path x is outside the allowed workspace y", "Blocked by workspace")).toBe(
      "Blocked by workspace"
    );
    expect(friendlyError(new Error("plain failure"), "Blocked by workspace")).toBe("plain failure");
  });
});
