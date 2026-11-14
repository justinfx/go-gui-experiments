<script lang="ts">
  import { onMount } from "svelte";
  import * as Table from "$lib/components/ui/table";
  import VirtualScroller, { type VItem } from "./VirtualScroller.svelte";
  import StockBar from "./cells/StockBar.svelte";
  import StatusBadge from "./cells/StatusBadge.svelte";
  import Stars from "./cells/Stars.svelte";
  import { app } from "$lib/state.svelte";

  const ROW = 37;
  const HEAD = 40;
  type Col = { key: string; label: string; sort: string; width: number; align: "left" | "right" | "center" };
  let cols = $state<Col[]>([
    { key: "id", label: "ID", sort: "id", width: 80, align: "right" },
    { key: "sku", label: "SKU", sort: "sku", width: 110, align: "left" },
    { key: "name", label: "Name", sort: "name", width: 260, align: "left" },
    { key: "category", label: "Category", sort: "category", width: 110, align: "left" },
    { key: "price", label: "Price", sort: "price", width: 100, align: "right" },
    { key: "quantity", label: "Qty", sort: "quantity", width: 80, align: "right" },
    { key: "stock", label: "Stock", sort: "quantity", width: 120, align: "left" },
    { key: "status", label: "Status", sort: "status", width: 120, align: "left" },
    { key: "rating", label: "Rating", sort: "rating", width: 110, align: "left" },
    { key: "updated", label: "Updated", sort: "updated", width: 150, align: "left" },
    { key: "featured", label: "Featured", sort: "featured", width: 80, align: "center" },
  ]);
  const totalW = $derived(cols.reduce((a, c) => a + c.width, 0));
  const ta = (a: string) => (a === "right" ? "text-right" : a === "center" ? "text-center" : "text-left");

  let sc: VirtualScroller;
  const total = $derived(app.total);

  function label(c: Col) {
    return c.label + (app.sortBy === c.sort && c.key !== "stock" ? (app.desc ? " ▼" : " ▲") : "");
  }

  function drag(e: PointerEvent, c: Col) {
    e.stopPropagation();
    e.preventDefault();
    const x0 = e.clientX;
    const w0 = c.width;
    const move = (ev: PointerEvent) => (c.width = Math.max(40, w0 + ev.clientX - x0));
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  function key(e: KeyboardEvent) {
    if (e.key === "ArrowDown") (e.preventDefault(), app.moveSelection(1));
    else if (e.key === "ArrowUp") (e.preventDefault(), app.moveSelection(-1));
  }

  onMount(() => {
    app.active = {
      scrollToRow: (i) => sc.scrollToRow(i),
      ensureVisible: (i) => sc.ensureVisible(i),
      topRow: () => sc.topRow(),
      visibleRange: () => sc.visibleRange(),
      focus: () => sc.focus(),
      element: () => sc.element(),
    };
    sc.scrollToRow(app.initialTop());
    return () => (app.active = null);
  });
  $effect(() => {
    void app.queryGen;
    sc?.scrollToRow(0);
  });
  $effect(() => {
    const r = app.scrollRequest;
    if (r) sc?.scrollToRow(r.index);
  });
</script>

<VirtualScroller
  bind:this={sc}
  count={Math.max(0, total)}
  itemSize={ROW}
  headerHeight={HEAD}
  contentWidth="{totalW}px"
  class="h-full w-full"
  label="table"
  onvisible={(f, l) => app.setVisible(f, l)}
  onkey={key}
>
  {#snippet header()}
    <table class="text-sm" style="width:{totalW}px; table-layout:fixed">
      <colgroup>{#each cols as c (c.key)}<col style="width:{c.width}px" />{/each}</colgroup>
      <Table.Header>
        <Table.Row class="hover:bg-transparent">
          {#each cols as c (c.key)}
            <Table.Head class="relative p-0 {ta(c.align)}" style="height:{HEAD}px">
              <button class="h-full w-full px-2 {ta(c.align)} cursor-pointer" data-testid="th-{c.key}" onclick={() => app.sortByField(c.sort)}>{label(c)}</button>
              <span class="absolute top-0 right-0 h-full w-1.5 cursor-col-resize" role="separator" aria-orientation="vertical" onpointerdown={(e) => drag(e, c)}></span>
            </Table.Head>
          {/each}
        </Table.Row>
      </Table.Header>
    </table>
  {/snippet}
  {#snippet empty()}No items match.{/snippet}
  {#snippet children(items: VItem[])}
    {#if total > 0 && items.length}
      <table class="absolute left-0 text-sm" style="top:{items[0].y}px; width:{totalW}px; table-layout:fixed" data-testid="rows">
        <colgroup>{#each cols as c (c.key)}<col style="width:{c.width}px" />{/each}</colgroup>
        <Table.Body>
          {#each items as vi (vi.index)}
            {@const it = app.row(vi.index)}
            <Table.Row
              style="height:{ROW}px"
              data-state={it && it.id === app.selectedId ? "selected" : undefined}
              data-testid="row"
              data-index={vi.index}
              onclick={() => it && app.requestSelect(it, vi.index)}
            >
              {#if it}
                <Table.Cell class="py-0 text-right" data-col="id">{it.id}</Table.Cell>
                <Table.Cell class="py-0 truncate" data-col="sku">{it.sku}</Table.Cell>
                <Table.Cell class="py-0 truncate" data-col="name">{it.name}</Table.Cell>
                <Table.Cell class="py-0" data-col="category">{it.category}</Table.Cell>
                <Table.Cell class="py-0 text-right" data-col="price">{it.priceText}</Table.Cell>
                <Table.Cell class="py-0 text-right" data-col="quantity">{it.quantity}</Table.Cell>
                <Table.Cell class="py-0" data-col="stock"><StockBar fraction={it.stockFraction} low={it.stockLow} /></Table.Cell>
                <Table.Cell class="py-0" data-col="status"><StatusBadge status={it.status} /></Table.Cell>
                <Table.Cell class="py-0" data-col="rating"><Stars full={it.starsFull} half={it.starsHalf} empty={it.starsEmpty} text={it.ratingText} /></Table.Cell>
                <Table.Cell class="py-0" data-col="updated">{it.updatedText}</Table.Cell>
                <Table.Cell class="py-0 text-center" data-col="featured">{it.featured ? "✓" : ""}</Table.Cell>
              {:else}
                {#each cols as c (c.key)}<Table.Cell class="py-0" data-placeholder="1">…</Table.Cell>{/each}
              {/if}
            </Table.Row>
          {/each}
        </Table.Body>
      </table>
    {/if}
  {/snippet}
</VirtualScroller>
