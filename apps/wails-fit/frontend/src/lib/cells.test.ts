import { describe, expect, it } from "vitest";
import { render } from "@testing-library/svelte";
import StatusBadge from "./components/cells/StatusBadge.svelte";
import Stars from "./components/cells/Stars.svelte";
import StockBar from "./components/cells/StockBar.svelte";
import { palette } from "./palette.svelte";
import { FormModel } from "./form.svelte";

const pal = (bg: string) => ({
  active: { background: bg, text: "#111111" }, backordered: { background: "#222222", text: "#333333" }, discontinued: { background: "#444444", text: "#555555" },
  stockTrack: "#E5E7EB", stockFill: "#2563EB", stockFillLow: "#DC2626", starFilled: "#F59E0B", starEmpty: "#D1D5DB",
  featured: { background: "#1", text: "#2" }, tiles: [], tileText: "#FFFFFF",
});

describe("custom cells", () => {
  it("status badge uses the palette for the current theme", () => {
    palette.set = { light: pal("#DCFCE7"), dark: pal("#14532D") };
    palette.dark = false;
    const { getByTestId, unmount } = render(StatusBadge, { status: "Active" });
    expect(getByTestId("status-badge").textContent).toBe("Active");
    expect(getByTestId("status-badge").dataset.bg).toBe("#DCFCE7");
    unmount();
    palette.dark = true;
    const r = render(StatusBadge, { status: "Active" });
    expect(r.getByTestId("status-badge").dataset.bg).toBe("#14532D");
  });

  it("stars draw full, half and empty shapes", () => {
    const { container } = render(Stars, { full: 3, half: 1, empty: 1, text: "3.5" });
    const kinds = [...container.querySelectorAll("svg")].map((s) => s.dataset.kind);
    expect(kinds).toEqual(["full", "full", "full", "half", "empty"]);
    expect(container.textContent).toContain("3.5");
  });

  it("stock bar fill width equals the fraction", () => {
    palette.dark = false;
    const { getByTestId } = render(StockBar, { fraction: 0.432, low: false, quantity: 432 });
    expect(getByTestId("stock-fill").getAttribute("width")).toBe(String(80 * 0.432));
    expect(getByTestId("stock-bar").dataset.fill).toBe("#2563EB");
  });
});

describe("FormModel", () => {
  const item = { id: 1, sku: "SKU-0000001", name: "Brushed Steel Lantern", category: "Garden", priceCents: 1234, quantity: 5, status: "Active", rating: 4.3, featured: false, restockDate: null, updatedAt: "", notes: "", priceText: "$12.34", ratingText: "4.3", updatedText: "" } as never;
  it("tracks dirty state and resets", () => {
    const f = new FormModel();
    f.load(item);
    expect(f.dirty).toBe(false);
    f.name = "ab";
    expect(f.dirty).toBe(true);
    f.reset();
    expect(f.dirty).toBe(false);
    expect(f.toEdit().rating).toBe(4.3);
  });
});
