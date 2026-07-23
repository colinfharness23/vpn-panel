import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import PortalApp from "@/portal/PortalApp";

describe("portal mobile and right-to-left locales", () => {
  afterEach(() => {
    localStorage.removeItem("nova-locale");
    window.history.replaceState({}, "", "/");
    document.documentElement.dir = "";
    document.documentElement.lang = "";
  });

  it("keeps the application layout LTR while rendering Arabic account copy", async () => {
    localStorage.setItem("nova-locale", "ar-EG");
    window.history.replaceState({}, "", "/account?preview=design");

    const { container } = render(<PortalApp />);

    expect(await screen.findByText("المركز الشخصي")).toBeTruthy();
    const shell = container.querySelector<HTMLElement>(".portal-shell");
    expect(shell).not.toBeNull();
    expect(shell?.dir).toBe("ltr");
    expect(shell?.dataset.copyDirection).toBe("rtl");
    expect(document.documentElement.dir).toBe("ltr");
    expect(document.documentElement.lang).toBe("ar-EG");
    expect(screen.getAllByText("نظرة عامة").length).toBeGreaterThan(0);
  });
});
