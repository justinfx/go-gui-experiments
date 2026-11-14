<script lang="ts">
  import { onMount } from "svelte";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import { Label } from "$lib/components/ui/label";
  import * as Popover from "$lib/components/ui/popover";
  import * as Select from "$lib/components/ui/select";
  import { app, type Theme } from "$lib/state.svelte";
  import { AllCats } from "$lib/cats";

  let searchEl = $state<HTMLInputElement | null>(null);
  let goto = $state("");
  onMount(() => {
    app.focusSearchFn = () => {
      searchEl?.focus();
      searchEl?.select();
    };
  });
  const catLabel = $derived(app.categories.length ? `Categories (${app.categories.length})` : "Categories");
  const statusLabel = $derived(app.status || "All statuses");
  const themeLabel = $derived({ system: "System", light: "Light", dark: "Dark" }[app.theme]);
  function go() {
    app.goToRow(Number(goto));
  }
</script>

<div class="flex items-center gap-2 border-b p-2">
  <Input
    bind:ref={searchEl}
    class="w-70"
    style="width:280px"
    placeholder="Search name or SKU"
    value={app.searchInput}
    oninput={(e) => app.onSearchInput(e.currentTarget.value)}
    onkeydown={(e) => e.key === "Enter" && app.commitSearch()}
  />
  <Popover.Root>
    <Popover.Trigger>
      {#snippet child({ props })}<Button {...props} variant="outline" data-testid="categories">{catLabel}</Button>{/snippet}
    </Popover.Trigger>
    <Popover.Content class="w-52">
      <div class="flex flex-col gap-2">
        {#each AllCats as c (c)}
          <div class="flex items-center gap-2">
            <Checkbox id="cat-{c}" checked={app.categories.includes(c)} onCheckedChange={(v) => app.toggleCategory(c, !!v)} />
            <Label for="cat-{c}">{c}</Label>
          </div>
        {/each}
        <Button variant="ghost" size="sm" onclick={() => app.clearCategories()}>Clear</Button>
      </div>
    </Popover.Content>
  </Popover.Root>
  <Select.Root type="single" value={app.status} onValueChange={(v) => app.setStatus(v)}>
    <Select.Trigger class="w-40" data-testid="status-filter">{statusLabel}</Select.Trigger>
    <Select.Content>
      <Select.Item value="" label="All statuses">All statuses</Select.Item>
      {#each ["Active", "Backordered", "Discontinued"] as s (s)}<Select.Item value={s} label={s}>{s}</Select.Item>{/each}
    </Select.Content>
  </Select.Root>
  <Input type="number" class="w-28" placeholder="Go to row" bind:value={goto} onkeydown={(e) => e.key === "Enter" && go()} data-testid="goto" />
  <Button variant="outline" onclick={go}>Go</Button>
  <div class="inline-flex rounded-md border" role="group" aria-label="View">
    <Button variant={app.view === "table" ? "default" : "ghost"} class="rounded-r-none" onclick={() => app.setView("table")}>Table</Button>
    <Button variant={app.view === "cards" ? "default" : "ghost"} class="rounded-l-none" onclick={() => app.setView("cards")}>Cards</Button>
  </div>
  <div class="flex-1"></div>
  <Button variant="outline" disabled={app.exportRunning} onclick={() => app.startExport()}>Export CSV</Button>
  <Select.Root type="single" value={app.theme} onValueChange={(v) => app.setTheme(v as Theme)}>
    <Select.Trigger class="w-32" data-testid="theme">{themeLabel}</Select.Trigger>
    <Select.Content>
      <Select.Item value="system" label="System">System</Select.Item>
      <Select.Item value="light" label="Light">Light</Select.Item>
      <Select.Item value="dark" label="Dark">Dark</Select.Item>
    </Select.Content>
  </Select.Root>
</div>
