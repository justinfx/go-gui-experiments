<script lang="ts">
  import { palette } from "$lib/palette.svelte";
  import type { ItemView } from "$lib/api";
  import StatusBadge from "./cells/StatusBadge.svelte";
  import Stars from "./cells/Stars.svelte";
  import StockBar from "./cells/StockBar.svelte";
  import { AllCats } from "$lib/cats";
  let { item, selected, onclick }: { item: ItemView | undefined; selected: boolean; onclick: () => void } = $props();
  const tile = $derived(item ? (palette.current?.tiles?.[AllCats.indexOf(item.category)] ?? "#64748B") : "");
</script>

<div
  role="option"
  aria-selected={selected}
  tabindex="-1"
  data-testid="card"
  data-id={item?.id}
  data-placeholder={item ? undefined : "1"}
  {onclick}
  onkeydown={() => {}}
  class="bg-card text-card-foreground box-border flex h-24 w-full cursor-pointer items-center gap-3 rounded-xl p-3 {selected ? 'border-2 border-primary' : 'border border-border'}"
>
  {#if !item}
    <span class="font-bold">Loading…</span>
  {:else}
    <div class="flex size-12 shrink-0 items-center justify-center rounded-lg text-lg font-bold" style="background:{tile}; color:{palette.current?.tileText ?? '#FFFFFF'}">{item.categoryInitial}</div>
    <div class="min-w-0 flex-1">
      <div class="truncate font-bold">{item.name}</div>
      <div class="text-muted-foreground truncate text-xs">{item.sku} · {item.category}</div>
      <div class="text-muted-foreground truncate text-xs">{item.notesPreview}</div>
      <div class="mt-0.5 flex items-center gap-1">
        <StatusBadge status={item.status} />
        {#if item.featured}
          <span class="inline-flex h-5 items-center rounded-full px-2 text-xs font-medium" style="background:{palette.current?.featured.background}; color:{palette.current?.featured.text}">Featured</span>
        {/if}
      </div>
    </div>
    <div class="flex w-40 shrink-0 flex-col items-end gap-0.5 text-xs">
      <div class="text-lg font-bold">{item.priceText}</div>
      <Stars full={item.starsFull} half={item.starsHalf} empty={item.starsEmpty} text={item.ratingText} />
      <StockBar fraction={item.stockFraction} low={item.stockLow} label={item.stockLabel} labelFirst />
    </div>
  {/if}
</div>
