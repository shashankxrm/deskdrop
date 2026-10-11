import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PairPanel } from "./PairPanel";

describe("PairPanel", () => {
  it("starts a new chain", () => {
    const onStart = vi.fn();
    render(<PairPanel onStart={onStart} onJoin={() => {}} />);
    fireEvent.click(screen.getByRole("button", { name: /start a new pairing/i }));
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("join stays disabled until the code looks complete", () => {
    const onJoin = vi.fn();
    render(<PairPanel onStart={() => {}} onJoin={onJoin} />);
    const join = screen.getByRole("button", { name: /^join$/i });
    expect(join).toBeDisabled();
    fireEvent.change(screen.getByLabelText(/pairing code/i), { target: { value: "abcd-efgh" } });
    expect(join).toBeEnabled();
    fireEvent.click(join);
    expect(onJoin).toHaveBeenCalledWith("ABCD-EFGH");
  });
});
