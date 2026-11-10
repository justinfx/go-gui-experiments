<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Slider } from "$lib/components/ui/slider";
  import * as Select from "$lib/components/ui/select";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import * as Popover from "$lib/components/ui/popover";
  import { Calendar } from "$lib/components/ui/calendar";
  import { parseDate } from "@internationalized/date";
  import { app } from "$lib/state.svelte";
  import { AllCats } from "$lib/cats";

  const f = app.form;
  const off = $derived(!f.loaded);
  let calOpen = $state(false);
  const calValue = $derived(f.restock ? safeDate(f.restock) : undefined);
  function safeDate(s: string) {
    try {
      return parseDate(s);
    } catch {
      return undefined;
    }
  }
  const err = (k: string) => f.errors[k];
</script>

{#snippet error(k: string)}
  {#if err(k)}<p class="text-destructive text-xs" data-testid="err-{k}">{err(k)}</p>{/if}
{/snippet}

<div class="flex h-full flex-col gap-3 overflow-auto p-3" data-testid="detail">
  {#if off}<p class="text-muted-foreground text-sm">Select an item to edit.</p>{/if}
  <div class="grid grid-cols-[100px_1fr] items-start gap-x-3 gap-y-2">
    <Label>ID</Label><span data-testid="f-id">{f.loaded?.id ?? ""}</span>
    <Label>SKU</Label><span data-testid="f-sku">{f.loaded?.sku ?? ""}</span>
    <Label for="f-name">Name</Label>
    <div><Input id="f-name" disabled={off} bind:value={f.name} />{@render error("name")}</div>
    <Label>Category</Label>
    <div>
      <Select.Root type="single" disabled={off} bind:value={f.category}>
        <Select.Trigger class="w-full" data-testid="f-category">{f.category}</Select.Trigger>
        <Select.Content>{#each AllCats as c (c)}<Select.Item value={c} label={c}>{c}</Select.Item>{/each}</Select.Content>
      </Select.Root>
    </div>
    <Label for="f-price">Price</Label>
    <div><Input id="f-price" disabled={off} bind:value={f.price} />{@render error("priceCents")}</div>
    <Label for="f-qty">Quantity</Label>
    <div><Input id="f-qty" type="number" min="0" max="1000000" step="1" disabled={off} bind:value={f.quantity} />{@render error("quantity")}</div>
    <Label>Status</Label>
    <div>
      <RadioGroup.Root class="flex flex-row gap-4" disabled={off} bind:value={f.status}>
        {#each ["Active", "Backordered", "Discontinued"] as s (s)}
          <div class="flex items-center gap-1.5"><RadioGroup.Item value={s} id="st-{s}" /><Label for="st-{s}">{s}</Label></div>
        {/each}
      </RadioGroup.Root>
      {@render error("status")}
    </div>
    <Label>Rating</Label>
    <div>
      <div class="flex items-center gap-3">
        <Slider type="single" min={0} max={50} step={1} disabled={off} bind:value={f.ratingTenths} data-testid="f-rating" />
        <span class="w-8 text-sm" data-testid="f-rating-text">{(f.ratingTenths / 10).toFixed(1)}</span>
      </div>
      {@render error("rating")}
    </div>
    <Label for="f-featured">Featured</Label>
    <div><Checkbox id="f-featured" disabled={off} bind:checked={f.featured} /></div>
    <Label>Restock date</Label>
    <div>
      <div class="flex items-center gap-2">
        <Popover.Root bind:open={calOpen}>
          <Popover.Trigger>
            {#snippet child({ props })}<Button {...props} variant="outline" disabled={off} class="w-40 justify-start" data-testid="f-restock">{f.restock || "Pick a date"}</Button>{/snippet}
          </Popover.Trigger>
          <Popover.Content class="w-auto p-0">
            <Calendar type="single" value={calValue} onValueChange={(v) => { f.restock = v ? v.toString() : ""; calOpen = false; }} />
          </Popover.Content>
        </Popover.Root>
        <Button variant="ghost" disabled={off} onclick={() => (f.restock = "")}>Clear</Button>
      </div>
      {@render error("restockDate")}
    </div>
    <Label>Updated</Label><span data-testid="f-updated">{f.loaded?.updatedText ?? ""}</span>
    <Label for="f-notes">Notes</Label>
    <div><Textarea id="f-notes" rows={4} disabled={off} bind:value={f.notes} />{@render error("notes")}</div>
  </div>
  <div class="flex items-center gap-2">
    <Button disabled={!f.dirty || f.saving} onclick={() => app.save()}>Save</Button>
    <Button variant="outline" disabled={!f.dirty} onclick={() => app.revert()}>Revert</Button>
    <Button variant="outline" disabled={off} onclick={() => app.copyRow()}>Copy Row</Button>
    <Button variant="outline" disabled={off} onclick={() => app.importNotes()}>Import Notes…</Button>
    {#if f.dirty}<span class="text-muted-foreground text-sm">Unsaved changes</span>{/if}
  </div>
</div>
