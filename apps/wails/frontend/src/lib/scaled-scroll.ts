// Scroll position mapping for lists that are taller than a browser can
// scroll. Browsers (Chrome and WebKit) cap an element's height at about 33.5
// million pixels; one million table rows of 37 px, or one million cards of
// 104 px, are taller. When the virtual height exceeds MAX_DOM_PX the real
// scrollable height is clamped and scrollTop is scaled to the virtual offset.
//
// The virtualizer works entirely in virtual coordinates. A programmatic jump
// to an exact virtual offset records an anchor so the same offset is reported
// back when the scroll event arrives (one real pixel can be 30 or more rows).

export const MAX_DOM_PX = 30_000_000;

export class ScaledScroll {
  el: HTMLElement | null = null;
  /** Height of the sticky header inside the scroller. */
  headerPx = 0;
  virtualTotal = 0;
  /** The last reported (scrollTop, virtual offset) pair. */
  lastScrollTop = 0;
  lastVirtual = 0;
  /** Set by the virtualizer's offset observer. */
  report: ((offset: number, isScrolling: boolean) => void) | null = null;
  private anchor: { scrollTop: number; virtual: number } | null = null;

  /** Height of the content element in real pixels. */
  get domHeight(): number {
    return Math.min(this.virtualTotal, MAX_DOM_PX);
  }

  /** Visible body height. */
  viewport(): number {
    return Math.max(0, (this.el?.clientHeight ?? 0) - this.headerPx);
  }

  /** Largest virtual offset. */
  maxVirtual(): number {
    return Math.max(0, this.virtualTotal - this.viewport());
  }

  private factor(): number {
    const vp = this.viewport();
    const real = this.domHeight - vp;
    const virt = this.virtualTotal - vp;
    return real > 0 && virt > real ? virt / real : 1;
  }

  toVirtual(scrollTop: number): number {
    const a = this.anchor;
    if (a && Math.abs(scrollTop - a.scrollTop) <= 1) return a.virtual;
    this.anchor = null;
    return Math.min(this.maxVirtual(), scrollTop * this.factor());
  }

  toScroll(virtual: number): number {
    return virtual / this.factor();
  }

  /** The virtualizer's observeElementOffset. */
  observeOffset = (
    inst: { scrollElement: Element | null },
    cb: (offset: number, isScrolling: boolean) => void,
  ): (() => void) => {
    const el = inst.scrollElement as HTMLElement | null;
    if (!el) return () => {};
    this.report = cb;
    const handler = () => {
      const v = this.toVirtual(el.scrollTop);
      this.lastScrollTop = el.scrollTop;
      this.lastVirtual = v;
      cb(v, true);
    };
    const end = () => cb(this.lastVirtual, false);
    el.addEventListener("scroll", handler, { passive: true });
    let t: ReturnType<typeof setTimeout> | undefined;
    const endLater = () => {
      clearTimeout(t);
      t = setTimeout(end, 120);
    };
    el.addEventListener("scroll", endLater, { passive: true });
    handler();
    return () => {
      el.removeEventListener("scroll", handler);
      el.removeEventListener("scroll", endLater);
      clearTimeout(t);
      if (this.report === cb) this.report = null;
    };
  };

  /** The virtualizer's observeElementRect: the body viewport. */
  observeRect = (
    inst: { scrollElement: Element | null },
    cb: (rect: { width: number; height: number }) => void,
  ): (() => void) => {
    const el = inst.scrollElement as HTMLElement | null;
    if (!el) return () => {};
    const push = () => cb({ width: el.clientWidth, height: this.viewport() });
    push();
    const ro = new ResizeObserver(push);
    ro.observe(el);
    return () => ro.disconnect();
  };

  /** Scrolls so that virtual offset v is at the top of the body. */
  scrollToVirtual(v: number): void {
    const el = this.el;
    if (!el) return;
    v = Math.max(0, Math.min(this.maxVirtual(), v));
    el.scrollTop = this.toScroll(v);
    const actual = el.scrollTop;
    this.anchor = { scrollTop: actual, virtual: v };
    this.lastScrollTop = actual;
    this.lastVirtual = v;
    this.report?.(v, false);
  }

  /** Real top (in content coordinates) of an item that starts at virtual `start`. */
  renderTop(start: number): number {
    return this.lastScrollTop + (start - this.lastVirtual);
  }
}
