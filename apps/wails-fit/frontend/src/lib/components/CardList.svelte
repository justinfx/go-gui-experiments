<script lang="ts">
  import { onMount } from "svelte";
  import VirtualScroller, { type VItem } from "./VirtualScroller.svelte";
  import Card from "./Card.svelte";
  import { app } from "$lib/state.svelte";

  const SIZE = 104; // 96 px card + 8 px gap
  let sc: VirtualScroller;
  const total = $derived(app.total);

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

<VirtualScroller bind:this={sc} count={Math.max(0, total)} itemSize={SIZE} class="h-full w-full px-2" label="cards" onvisible={(f, l) => app.setVisible(f, l)} onkey={key}>
  {#snippet empty()}No items match.{/snippet}
  {#snippet children(items: VItem[])}
    {#each items as vi (vi.index)}
      {@const it = app.row(vi.index)}
      <div class="absolute right-0 left-0" style="top:{vi.y}px; height:96px">
        <Card item={it} selected={!!it && it.id === app.selectedId} onclick={() => it && app.requestSelect(it, vi.index)} />
      </div>
    {/each}
  {/snippet}
</VirtualScroller>
