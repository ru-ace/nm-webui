<script lang="ts">
  import { ChevronDown, ChevronUp, Copy, Trash2, ToggleLeft, ToggleRight, Loader2, Wifi, Cable, Database, Settings } from 'lucide-svelte';
  import { activateConnection, deactivateConnection, toggleAutoconnect, forgetConnection } from '$lib/stores/app';

  export let conn: any;
  export let loadingMap: any;
  export let onEdit: ((c: any) => void) | undefined = undefined;

  let expanded = false;

  function getTypeIconType() {
    if (conn.is_wifi) return 'wifi';
    if (conn.type_name === 'ethernet') return 'cable';
    return 'database';
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
  }
</script>

<div class="card bg-base-100 shadow-sm border border-base-300 card-hover">
  <div class="card-body p-4">
    <div class="flex items-start justify-between gap-4">
      <div class="flex items-center gap-3 flex-1 min-w-0">
        <div class="p-2 bg-primary/10 rounded-lg">
          <svelte:component this={getTypeIconType() === 'wifi' ? Wifi : getTypeIconType() === 'cable' ? Cable : Database} class="w-4 h-4" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <h3 class="font-semibold truncate">{conn.id}</h3>
            {#if conn.active}
              <span class="badge badge-success badge-sm">Active</span>
            {/if}
            {#if conn.is_wifi && conn.ssid}
              <span class="badge badge-ghost badge-sm">"{conn.ssid}"</span>
            {/if}
          </div>
          <p class="text-sm text-base-content/60 truncate">
            {conn.interface || 'Any interface'} • {conn.ipv4_method} / {conn.ipv6_method}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <button
          class="btn btn-ghost btn-sm gap-1"
          onclick={() => toggleAutoconnect(conn.uuid, !conn.autoconnect)}
          disabled={loadingMap[`toggle-${conn.uuid}`]}
        >
          {#if conn.autoconnect}
            <ToggleRight class="w-4 h-4" />
          {:else}
            <ToggleLeft class="w-4 h-4" />
          {/if}
          <span class="hidden sm:inline">{conn.autoconnect ? 'Auto' : 'Manual'}</span>
        </button>
        {#if conn.active}
          <button
            class="btn btn-error btn-sm gap-1"
            onclick={() => deactivateConnection(conn.uuid)}
            disabled={loadingMap[`deactivate-${conn.uuid}`]}
          >
            Disconnect
          </button>
        {:else if conn.device}
          <button
            class="btn btn-primary btn-sm gap-1"
            onclick={() => activateConnection(conn.uuid)}
            disabled={loadingMap[`activate-${conn.uuid}`]}
          >
            Connect
          </button>
        {/if}
<button class="btn btn-ghost btn-sm" onclick={() => expanded = !expanded}>
            {#if expanded}
              <ChevronUp class="w-4 h-4" />
            {:else}
              <ChevronDown class="w-4 h-4" />
            {/if}
          </button>
      </div>
    </div>

    {#if expanded}
      <div class="mt-4 pt-4 border-t border-base-300 space-y-3">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
          <div class="md:col-span-2"><span class="text-base-content/60">UUID</span><br>
            <div class="flex items-center gap-2">
              <span class="font-mono text-xs truncate flex-1">{conn.uuid}</span>
              <button class="btn btn-ghost btn-xs" onclick={() => copyToClipboard(conn.uuid)}><Copy class="w-3 h-3" /></button>
            </div>
          </div>
          <div><span class="text-base-content/60">Type</span><br><span>{conn.type_name}</span></div>
          <div><span class="text-base-content/60">Autoconnect</span><br><span>{conn.autoconnect ? 'Yes' : 'No'}</span></div>
          <div><span class="text-base-content/60">IPv4 Method</span><br><span class="badge badge-ghost">{conn.ipv4_method}</span></div>
          <div><span class="text-base-content/60">IPv6 Method</span><br><span class="badge badge-ghost">{conn.ipv6_method}</span></div>
          {#if conn.static4}
            <div class="md:col-span-2"><span class="text-base-content/60">Static IPv4</span><br>
              <span class="font-mono text-xs">{conn.static4.address}/{conn.static4.prefix}</span>
              {#if conn.static4.gateway} <span class="text-base-content/60">via {conn.static4.gateway}</span> {/if}
            </div>
          {/if}
          {#if conn.static6}
            <div class="md:col-span-2"><span class="text-base-content/60">Static IPv6</span><br>
              <span class="font-mono text-xs">{conn.static6.address}/{conn.static6.prefix}</span>
            </div>
          {/if}
        </div>
        <div class="flex gap-2 pt-2 border-t border-base-300">
          <button class="btn btn-ghost btn-sm flex-1 gap-1" onclick={() => { if (onEdit) { onEdit(conn); } else { window.dispatchEvent(new CustomEvent('edit-connection', { detail: conn })); } }}>
            <Settings class="w-4 h-4" /> Edit
          </button>
          <button class="btn btn-error btn-sm flex-1 gap-1" onclick={() => forgetConnection(conn.uuid)} disabled={loadingMap[`delete-${conn.uuid}`]}>
            <Trash2 class="w-4 h-4" /> Forget
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>