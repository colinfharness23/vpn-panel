import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import PortalApp from "@/portal/PortalApp";

describe("portal residential relay", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    localStorage.removeItem("nova-locale");
    window.history.replaceState({}, "", "/");
  });

  it("offers AnyTLS, submits once, and renders the saved relay immediately", async () => {
    localStorage.setItem("nova-locale", "zh-CN");
    window.history.replaceState({}, "", "/subscription?preview=design");
    let releaseSave: (() => void) | undefined;
    const saveGate = new Promise<void>((resolve) => {
      releaseSave = resolve;
    });
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path.endsWith("/api/v1/passport/csrf-token")) {
          return new Response(
            JSON.stringify({ success: true, msg: "", obj: "csrf-token" }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        if (
          path.endsWith("/api/v1/user/residential-relays") &&
          init?.method === "POST"
        ) {
          await saveGate;
          return new Response(
            JSON.stringify({
              success: true,
              msg: "",
              obj: {
                enabled: true,
                limit: 2,
                lines: [
                  { id: 101, name: "新加坡高速线路", protocol: "vless" },
                  { id: 102, name: "日本优化线路", protocol: "trojan" },
                  { id: 103, name: "AnyTLS 线路", protocol: "anytls" },
                ],
                relays: [
                  {
                    id: "relay-london",
                    inboundId: 101,
                    lineName: "新加坡高速线路",
                    protocol: "vless",
                    name: "伦敦",
                    host: "8.8.8.8",
                    port: 1080,
                    username: "relay-user",
                    hasPassword: true,
                    status: "active",
                    createdAt: "2026-07-24T00:00:00Z",
                    links: ["vless://example#%E4%BC%A6%E6%95%A6"],
                  },
                ],
              },
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return new Response(
          JSON.stringify({ success: false, msg: "unexpected request", obj: null }),
          { status: 500, headers: { "Content-Type": "application/json" } },
        );
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    render(<PortalApp />);
    fireEvent.click(
      await screen.findByRole("button", { name: /添加中转/ }),
    );
    const dialog = await screen.findByRole("dialog");

    const lineSelect = within(dialog).getByRole("combobox", { name: "中转线路" });
    fireEvent.mouseDown(lineSelect);
    expect(await screen.findByText("AnyTLS 线路 · ANYTLS")).toBeTruthy();
    fireEvent.change(within(dialog).getByRole("textbox", { name: "配置名称" }), {
      target: { value: "伦敦" },
    });
    fireEvent.change(within(dialog).getByRole("textbox", { name: "SOCKS5 主机或 IP" }), {
      target: { value: "8.8.8.8" },
    });
    fireEvent.change(within(dialog).getByRole("textbox", { name: "用户名（可选）" }), {
      target: { value: "relay-user" },
    });
    fireEvent.change(within(dialog).getByLabelText("密码（可选）"), {
      target: { value: "relay-password" },
    });

    const save = within(dialog).getByRole("button", { name: "保存并应用" });
    fireEvent.click(save);
    fireEvent.click(save);
    await waitFor(() => {
      const posts = fetchMock.mock.calls.filter(
        ([path, init]) =>
          String(path).endsWith("/api/v1/user/residential-relays") &&
          init?.method === "POST",
      );
      expect(posts).toHaveLength(1);
    });

    releaseSave?.();
    expect(await screen.findByText("伦敦")).toBeTruthy();
    expect(screen.getByText("已配置 1/2")).toBeTruthy();
  });
});
