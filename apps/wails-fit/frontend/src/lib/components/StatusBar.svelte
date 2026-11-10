<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Progress } from "$lib/components/ui/progress";
  import { app } from "$lib/state.svelte";
  const st = $derived(app.stats());
  const total = $derived(app.total);
  const pct = $derived(app.exportTotal > 0 ? Math.floor((app.exportDone * 100) / app.exportTotal) : 0);
</script>

<div class="flex h-9 items-center gap-3 border-t px-3 text-sm" data-testid="statusbar">
  <span data-testid="status-left">
    {total >= 0 ? app.totalText : "…"} items{total > 0 ? ` · rows ${app.visFirst + 1}-${Math.min(total, app.visLast + 1)}` : ""} · last fetch {Math.round(st.lastFetch)} ms · {st.cachedPages} pages cached
  </span>
  {#if app.message}<span class="text-muted-foreground" data-testid="message">{app.message}</span>{/if}
  <div class="flex-1"></div>
  {#if app.exportRunning}
    <div class="flex items-center gap-2" data-testid="export-progress">
      <Progress value={pct} max={100} class="w-50" style="width:200px" />
      <span class="w-10">{pct}%</span>
      <Button size="sm" variant="outline" onclick={() => app.cancelExport()}>Cancel</Button>
    </div>
  {/if}
  <span data-testid="identity">{app.host?.identity ?? "unknown user"}</span>
  <Button size="sm" variant="outline" onclick={() => app.openHost()}>Host</Button>
</div>
