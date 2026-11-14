<script lang="ts">
  import * as Dialog from "$lib/components/ui/dialog";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { Button } from "$lib/components/ui/button";
  import { app } from "$lib/state.svelte";
  import { UIMarker } from "$lib/marker";

  let revealErr = $state("");
  async function reveal(w: "config" | "export") {
    revealErr = (await app.reveal(w)) ?? "";
  }
  const h = $derived(app.host);
  const tip = "Not available in the browser";
</script>

<Dialog.Root bind:open={app.hostOpen}>
  <Dialog.Content class="max-w-xl" data-testid="host-dialog">
    <Dialog.Header><Dialog.Title>Host Info</Dialog.Title></Dialog.Header>
    {#if h}
      <dl class="grid grid-cols-[140px_1fr] gap-x-3 gap-y-1 text-sm">
        {#each h.rows as r (r.label)}
          <dt class="text-muted-foreground">{r.label}</dt>
          <dd class="break-all" data-testid="host-{r.label}">{r.value}</dd>
        {/each}
      </dl>
      {#each h.notes as n (n)}<p class="text-muted-foreground text-xs">{n}</p>{/each}
      {#if revealErr}<p class="text-destructive text-xs">{revealErr}</p>{/if}
      <Dialog.Footer>
        <span title={h.actions ? "" : tip}><Button variant="outline" disabled={!h.actions} title={h.actions ? "" : tip} onclick={() => reveal("config")}>Reveal Config Folder</Button></span>
        <span title={h.actions ? "" : tip}><Button variant="outline" disabled={!h.actions || !app.hasExport} title={h.actions ? "" : tip} onclick={() => reveal("export")}>Reveal Last Export</Button></span>
        <Dialog.Close>{#snippet child({ props })}<Button {...props}>Close</Button>{/snippet}</Dialog.Close>
      </Dialog.Footer>
    {/if}
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={app.aboutOpen}>
  <Dialog.Content data-testid="about-dialog">
    <Dialog.Header><Dialog.Title>About GGUIX</Dialog.Title></Dialog.Header>
    <div class="text-sm">
      <div>GGUIX - MyGo</div>
      <div>{app.info?.framework}</div>
      <div>{app.info?.goVersion}</div>
      <div>BuildMarker {app.info?.buildMarker}</div>
      <div>UIMarker {UIMarker}</div>
    </div>
  </Dialog.Content>
</Dialog.Root>

<AlertDialog.Root open={!!app.confirm} onOpenChange={(o) => !o && (app.confirm = null)}>
  <AlertDialog.Content data-testid="confirm-dialog">
    <AlertDialog.Header><AlertDialog.Title>{app.confirm?.title}</AlertDialog.Title></AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>{app.confirm?.cancelLabel}</AlertDialog.Cancel>
      <AlertDialog.Action onclick={() => { const c = app.confirm; app.confirm = null; c?.onConfirm(); }}>{app.confirm?.confirmLabel}</AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<AlertDialog.Root open={!!app.errorDialog} onOpenChange={(o) => !o && (app.errorDialog = null)}>
  <AlertDialog.Content data-testid="error-dialog">
    <AlertDialog.Header>
      <AlertDialog.Title>{app.errorDialog?.title}</AlertDialog.Title>
      <AlertDialog.Description>{app.errorDialog?.message}</AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer><AlertDialog.Action>OK</AlertDialog.Action></AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
