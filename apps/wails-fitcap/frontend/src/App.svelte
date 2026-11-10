<script lang="ts">
  import { onMount } from "svelte";
  import { ModeWatcher, mode } from "mode-watcher";
  import { Toaster } from "$lib/components/ui/sonner";
  import { app } from "$lib/state.svelte";
  import { palette } from "$lib/palette.svelte";
  import { api } from "$lib/api";
  import Toolbar from "$lib/components/Toolbar.svelte";
  import DataTable from "$lib/components/DataTable.svelte";
  import CardList from "$lib/components/CardList.svelte";
  import DetailForm from "$lib/components/DetailForm.svelte";
  import StatusBar from "$lib/components/StatusBar.svelte";
  import Dialogs from "$lib/components/Dialogs.svelte";
  import WebMenuBar from "$lib/components/web/WebMenuBar.svelte";
  import { runBench } from "./bench/bench";

  let fileEl = $state<HTMLInputElement | null>(null);
  let split = $state(65);
  let box = $state<HTMLDivElement | undefined>();

  onMount(async () => {
    await app.init();
    app.importFileFn = () => fileEl?.click();
    if (app.bench) void runBench();
  });

  $effect(() => {
    palette.dark = mode.current === "dark";
  });
  $effect(() => {
    void [app.theme, app.view, app.sortBy, app.desc, app.search, app.categories.length, app.status, app.selectedId];
    app.schedulePrefsSave();
  });
  $effect(() => {
    if (!app.isDesktop) return;
    void api.SetMenuState({ view: app.view, theme: app.theme, exportRunning: app.exportRunning, hasExport: app.hasExport, hasSelection: !!app.form.loaded });
  });

  // gguix:target web
  function keydown(e: KeyboardEvent) {
    if (!app.isWeb || !(e.ctrlKey || e.metaKey)) return;
    const k = e.key.toLowerCase();
    const map: Record<string, string> = { e: "export", o: "import-notes", "1": "view-table", "2": "view-cards", r: "reload", f: "focus-search", i: "host-info" };
    const act = e.shiftKey && k === "c" ? "copy-row" : e.shiftKey ? "" : map[k];
    if (act) {
      e.preventDefault();
      void app.action(act);
    }
  }
  // gguix:end

  function dragSplit(e: PointerEvent) {
    e.preventDefault();
    const move = (ev: PointerEvent) => {
      if (!box) return;
      const r = box.getBoundingClientRect();
      split = Math.max(30, Math.min(85, ((ev.clientX - r.left) / r.width) * 100));
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }
</script>

<svelte:window onkeydown={keydown} />
<ModeWatcher defaultMode="system" />
<Toaster />
<input bind:this={fileEl} type="file" accept=".txt" class="hidden" data-testid="notes-file" onchange={(e) => { const f = e.currentTarget.files?.[0]; if (f) void app.importNotesFile(f); e.currentTarget.value = ""; }} />

{#if app.ready}
  <div class="bg-background text-foreground flex h-screen flex-col">
    <!-- gguix:target web -->
    {#if app.isWeb}<WebMenuBar />{/if}
    <!-- gguix:end -->
    <Toolbar />
    <div class="flex min-h-0 flex-1" bind:this={box}>
      <div class="min-w-0" style="width:{split}%">
        {#if app.view === "table"}<DataTable />{:else}<CardList />{/if}
      </div>
      <div class="bg-border w-1 cursor-col-resize" role="separator" aria-orientation="vertical" onpointerdown={dragSplit}></div>
      <div class="min-w-0 flex-1"><DetailForm /></div>
    </div>
    <StatusBar />
  </div>
  <Dialogs />
{/if}
