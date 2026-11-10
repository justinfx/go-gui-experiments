<script lang="ts">
  import { palette } from "$lib/palette.svelte";
  let { full, half, empty, text, size = 12 }: { full: number; half: number; empty: number; text: string; size?: number } = $props();
  const filled = $derived(palette.current?.starFilled ?? "currentColor");
  const emptyC = $derived(palette.current?.starEmpty ?? "currentColor");
  const kinds = $derived([...Array(full).fill("full"), ...Array(half).fill("half"), ...Array(empty).fill("empty")] as string[]);
  // Five-point star outline in a 24 x 24 box.
  const STAR = "M12 2 L14.94 8.62 L22 9.27 L16.7 14.14 L18.18 21.02 L12 17.5 L5.82 21.02 L7.3 14.14 L2 9.27 L9.06 8.62 Z";
  const uid = Math.random().toString(36).slice(2, 8);
</script>

<!-- Stars: Stars(rating) gives full, half and empty counts; a half star has its left half filled. -->
<span class="inline-flex items-center gap-0.5 whitespace-nowrap" data-testid="stars" data-full={full} data-half={half} data-empty={empty}>
  <span class="inline-flex">
    {#each kinds as kind, i (i)}
      <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" data-kind={kind}>
        {#if kind === "half"}
          <defs>
            <clipPath id="half-{uid}-{i}"><rect x="0" y="0" width="12" height="24" /></clipPath>
          </defs>
          <path d={STAR} fill={emptyC} />
          <path d={STAR} fill={filled} clip-path="url(#half-{uid}-{i})" />
        {:else}
          <path d={STAR} fill={kind === "full" ? filled : emptyC} />
        {/if}
      </svg>
    {/each}
  </span>
  <span class="ml-1">{text}</span>
</span>
