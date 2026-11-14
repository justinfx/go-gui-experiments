<script lang="ts">
  import { palette } from "$lib/palette.svelte";
  let { status }: { status: string } = $props();
  const c = $derived(palette.badge(status));
</script>

<!-- Status badge: a pill in the 4.5.1 colors for the current theme. -->
<span
  data-testid="status-badge"
  data-bg={c.background}
  data-fg={c.text}
  class="inline-flex h-5 items-center rounded-full px-2 text-xs font-medium whitespace-nowrap"
  style="background-color:{c.background}; color:{c.text}">{status}</span
>
