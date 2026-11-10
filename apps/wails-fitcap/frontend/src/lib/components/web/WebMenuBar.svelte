<script lang="ts">
  // Browser stand-in for the native menu bar (web target only).
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
  import { Button } from "$lib/components/ui/button";
  import { app } from "$lib/state.svelte";
  type Item = [string, string, string?];
  const menus: [string, Item[]][] = [
    ["File", [["Export CSV…", "export", "Ctrl+E"], ["Import Notes…", "import-notes", "Ctrl+O"]]],
    ["Edit", [["Copy Row", "copy-row", "Ctrl+Shift+C"]]],
    ["View", [["Table", "view-table", "Ctrl+1"], ["Cards", "view-cards", "Ctrl+2"], ["Theme: System", "theme-system"], ["Theme: Light", "theme-light"], ["Theme: Dark", "theme-dark"], ["Reload", "reload", "Ctrl+R"], ["Focus Search", "focus-search", "Ctrl+F"]]],
    ["Help", [["About GGUIX", "about"], ["Host Info…", "host-info", "Ctrl+I"]]],
  ];
</script>

<div class="flex items-center gap-1 border-b px-2 py-1" data-testid="web-menubar">
  {#each menus as [title, items] (title)}
    <DropdownMenu.Root>
      <DropdownMenu.Trigger>{#snippet child({ props })}<Button {...props} variant="ghost" size="sm">{title}</Button>{/snippet}</DropdownMenu.Trigger>
      <DropdownMenu.Content>
        {#each items as [label, act, key] (act)}
          <DropdownMenu.Item onclick={() => app.action(act)}>{label}{#if key}<DropdownMenu.Shortcut>{key}</DropdownMenu.Shortcut>{/if}</DropdownMenu.Item>
        {/each}
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  {/each}
</div>
