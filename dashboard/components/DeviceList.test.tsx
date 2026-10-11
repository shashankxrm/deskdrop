import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DeviceList } from "./DeviceList";

describe("DeviceList", () => {
  it("lists devices and unpairs one", () => {
    const onRevoke = vi.fn();
    render(
      <DeviceList
        onRevoke={onRevoke}
        devices={[
          { id: "lap-1", name: "Mac", type: "laptop", created_at: "2026-01-01T00:00:00Z" },
          { id: "web-1", name: "Dashboard", type: "web", created_at: "2026-01-01T00:00:00Z" },
        ]}
      />,
    );
    expect(screen.getByText("Mac")).toBeInTheDocument();
    expect(screen.getByText("laptop")).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: /unpair/i })[0]);
    expect(onRevoke).toHaveBeenCalledWith("lap-1");
  });
});
