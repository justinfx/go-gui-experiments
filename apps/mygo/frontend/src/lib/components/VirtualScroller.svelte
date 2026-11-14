<script lang="ts" module>
  export interface VItem {
    index: number;
    /** Top in content coordinates (real pixels, below the sticky header). */
    y: number;
  }
</script>

<script lang="ts">
  // A virtualized scroller built on @tanstack/svelte-virtual. The scroll
  // position mapping in scaled-scroll.ts lets it handle lists taller than
  // the browser's maximum element height.
  import type { Snippet } from "svelte";
  import { untrack } from "svelte";
  import { get } from "svelte/store";
  import { createVirtualizer } from "@tanstack/svelte-virtual";
  import { ScaledScroll, MAX_DOM_PX } from "$lib/scaled-scroll";

  let {
    count,
    itemSize,
    headerHeight = 0,
    overscan = 6,
    contentWidth = "100%",
    class: className = "",
    label = "",
    onvisible,
    onkey,
    header,
    children,
    empty,
  }: {
    count: number;
    itemSize: number;
    headerHeight?: number;
    overscan?: number;
    contentWidth?: string;
    class?: string;
    label?: string;
    onvisible?: (first: number, last: number) => void;
    onkey?: (e: KeyboardEvent) => void;
    header?: Snippet;
    children: Snippet<[VItem[]]>;
    empty?: Snippet;
  } = $props();

  const scroll = new ScaledScroll();
  const domHeight = $derived(Math.min(count * itemSize, MAX_DOM_PX));
  let el = $state<HTMLDivElement | null>(null);

  const virtualizer = createVirtualizer<HTMLDivElement, Element>({
    count: 0,
    getScrollElement: () => el,
    estimateSize: () => itemSize,
    overscan: 6,
    observeElementOffset: scroll.observeOffset as never,
    observeElementRect: scroll.observeRect as never,
    scrollToFn: ((offset: number) => scroll.scrollToVirtual(offset)) as never,
  });

  $effect(() => {
    scroll.el = el;
    scroll.headerPx = headerHeight;
    scroll.virtualTotal = count * itemSize;
    const c = count;
    const size = itemSize;
    const o = overscan;
    const e = el;
    untrack(() => get(virtualizer).setOptions({ count: c, estimateSize: () => size, overscan: o, getScrollElement: () => e }));
  });

  const items = $derived.by<VItem[]>(() => {
    const v = $virtualizer;
    // Reading these makes the derivation depend on the scroll state.
    void v.scrollOffset;
    return v.getVirtualItems().map((vi) => ({ index: vi.index, y: scroll.renderTop(vi.start) }));
  });

  let lastFirst = -1;
  let lastLast = -2;
  $effect(() => {
    const v = $virtualizer;
    const off = v.scrollOffset ?? 0;
    const vp = scroll.viewport();
    void count;
    const first = Math.min(Math.max(0, Math.floor(off / itemSize)), Math.max(0, count - 1));
    const last = Math.max(first, Math.min(count - 1, Math.ceil((off + vp) / itemSize) - 1));
    if (first !== lastFirst || last !== lastLast) {
      lastFirst = first;
      lastLast = last;
      onvisible?.(first, last);
    }
  });

  // ---- exported API ----

  /** Makes row `index` the top visible row. */
  export function scrollToRow(index: number): void {
    scroll.scrollToVirtual(index * itemSize);
  }

  /** Scrolls the minimum needed to show row `index` fully. */
  export function ensureVisible(index: number): void {
    const top = index * itemSize;
    const bottom = top + itemSize;
    const off = scroll.lastVirtual;
    const vp = scroll.viewport();
    if (top < off) scroll.scrollToVirtual(top);
    else if (bottom > off + vp) scroll.scrollToVirtual(bottom - vp);
  }

  /** The first (top) visible row index. */
  export function topRow(): number {
    return Math.floor(scroll.lastVirtual / itemSize);
  }

  export function visibleRange(): [number, number] {
    const off = scroll.lastVirtual;
    const first = Math.floor(off / itemSize);
    const last = Math.max(first, Math.min(count - 1, Math.ceil((off + scroll.viewport()) / itemSize) - 1));
    return [first, last];
  }

  export function focus(): void {
    el?.focus();
  }

  export function element(): HTMLElement | null {
    return el;
  }

  function keydown(e: KeyboardEvent) {
    onkey?.(e);
  }
</script>

<div
  bind:this={el}
  class="relative overflow-auto outline-none {className}"
  tabindex="0"
  role="grid"
  aria-label={label}
  aria-rowcount={count}
  data-testid={label}
  onkeydown={keydown}
>
  {#if header}
    <div class="bg-background sticky top-0 z-10" style="width:{contentWidth}; min-width:100%; height:{headerHeight}px">
      {@render header()}
    </div>
  {/if}
  <div class="relative" style="height:{domHeight}px; width:{contentWidth}; min-width:100%">
    {@render children(items)}
  </div>
  {#if count === 0 && empty}
    <div class="absolute inset-x-0 flex items-center justify-center text-sm text-muted-foreground" style="top:{headerHeight}px; bottom:0">
      {@render empty()}
    </div>
  {/if}
</div>
