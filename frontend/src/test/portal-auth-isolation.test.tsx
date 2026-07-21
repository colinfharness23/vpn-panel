import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import PortalApp from "@/portal/PortalApp";

describe("portal authentication isolation", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders only the authentication dialog and requests no anonymous business bootstrap", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path.endsWith("/api/v1/guest/auth-config")) {
        return new Response(
          JSON.stringify({
            success: true,
            msg: "",
            obj: {
              site: {
                siteName: "NOVA",
                registrationClosed: "false",
                emailVerification: "false",
              },
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response(
        JSON.stringify({ success: false, msg: "unauthorized", obj: null }),
        { status: 401, headers: { "Content-Type": "application/json" } },
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    const { container } = render(<PortalApp />);

    expect(await screen.findByRole("dialog")).toBeTruthy();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock.mock.calls.some(([path]) => String(path).includes("/api/v1/user/dashboard"))).toBe(false);
    expect(container.querySelector(".portal-header")).toBeNull();
    expect(container.querySelector(".portal-main")).toBeNull();
    expect(container.querySelector(".portal-footer")).toBeNull();
    expect(container.querySelector(".telegram-support-fab")).toBeNull();
    expect(fetchMock.mock.calls.some(([path]) => String(path).includes("/api/v1/guest/bootstrap"))).toBe(false);
    expect(fetchMock.mock.calls.some(([path]) => String(path).includes("/api/v1/guest/applications"))).toBe(false);
  });
});
