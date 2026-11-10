<script lang="ts">
  import { palette } from "$lib/palette.svelte";
  let { fraction, low, label, quantity, labelFirst = false }: { fraction: number; low: boolean; label?: string; quantity?: number; labelFirst?: boolean } = $props();
  const track = $derived(palette.current?.stockTrack ?? "currentColor");
  const fill = $derived(palette.current ? (low ? palette.current.stockFillLow : palette.current.stockFill) : "currentColor");
</script>

<!-- Stock bar: an 80 x 8 px track with a fill of width StockFraction(quantity). -->
<span class="inline-flex items-center gap-2 whitespace-nowrap">
  {#if labelFirst && label}<span>{label}</span>{/if}
  <svg width="80" height="8" viewBox="0 0 80 8" role="img" aria-label="Stock level" data-testid="stock-bar" data-fraction={fraction} data-fill={fill} data-track={track}>
    <rect x="0" y="0" width="80" height="8" rx="4" fill={track} />
    <rect x="0" y="0" width={80 * fraction} height="8" rx="4" fill={fill} data-testid="stock-fill" />
  </svg>
  {#if !labelFirst && quantity !== undefined}<span>{quantity}</span>{/if}
</span>
