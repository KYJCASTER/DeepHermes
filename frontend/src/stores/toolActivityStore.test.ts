import { beforeEach, describe, expect, it, vi } from "vitest";
import { useToolActivityStore } from "./toolActivityStore";

describe("toolActivityStore", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T08:00:00Z"));
    useToolActivityStore.setState({
      items: [],
      isOpen: false,
    });
  });

  it("exports a tab-separated audit log with rollback status", () => {
    const store = useToolActivityStore.getState();

    store.addCall({
      id: "call-1",
      toolName: "write_file",
      arguments: "{\"file_path\":\"README.md\"}",
      risk: "write",
    });
    store.finishCall({
      toolCallId: "call-1",
      success: true,
      rollbackAvailable: true,
      rollbackPath: "README.md",
    });
    store.markRolledBack("call-1", "restored");

    expect(useToolActivityStore.getState().exportAuditLog()).toBe(
      [
        "timestamp\tstatus\ttool\trisk\targuments",
        "2026-05-12T08:00:00.000Z\trolled_back\twrite_file\twrite\t{\"file_path\":\"README.md\"}",
      ].join("\n")
    );
  });

  it("keeps the newest 80 activities", () => {
    for (let i = 0; i < 85; i++) {
      useToolActivityStore.getState().addCall({
        id: `call-${i}`,
        toolName: "read_file",
        arguments: "{}",
        risk: "read",
      });
    }

    const items = useToolActivityStore.getState().items;
    expect(items).toHaveLength(80);
    expect(items[0].id).toBe("call-84");
    expect(items[79].id).toBe("call-5");
  });
});
