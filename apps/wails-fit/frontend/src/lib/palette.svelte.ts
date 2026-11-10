import type { models } from "./api";

type PaletteView = models.PaletteView;

// The custom-rendering colors come from the Go side (core.Colors) so they
// are exactly the section 4.5.1 values. Until they load, light values from
// the same source are not available, so cells render with currentColor.
class PaletteState {
  set = $state<models.PaletteSet | null>(null);
  dark = $state(false);
  get current(): PaletteView | null {
    if (!this.set) return null;
    return this.dark ? this.set.dark : this.set.light;
  }
  badge(status: string): { background: string; text: string } {
    const p = this.current;
    if (!p) return { background: "transparent", text: "currentColor" };
    return status === "Backordered" ? p.backordered : status === "Discontinued" ? p.discontinued : p.active;
  }
}

export const palette = new PaletteState();
