<script lang="ts">
  import { X, Wifi, Cable, Database, Smartphone } from '@lucide/svelte';
  import { profileModal } from '$lib/stores/modals';
  import { navigate } from '$lib/stores/router';
  import type { ConnectionInfo } from '$lib/api/client';
  import { fly } from 'svelte/transition';

  function typeIcon(conn: ConnectionInfo) {
    if (conn.is_wifi) return Wifi;
    if (conn.is_modem) return Smartphone;
    if (conn.type_name === 'ethernet') return Cable;
    return Database;
  }

  function handleSelect(conn: ConnectionInfo) {
    $profileModal?.resolve(conn);
    close();
  }

  function handleManage() {
    const target = $profileModal?.manage;
    close();
    if (target) navigate(target);
  }

  function close() {
    $profileModal?.resolve(null);
    profileModal.set(null);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') close();
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if $profileModal}
  <div class="modal modal-open" role="dialog" tabindex="-1">
    <div class="modal-box relative" in:fly={{ y: 20, duration: 200 }} out:fly={{ y: -20, duration: 150 }}>
      <button
        type="button"
        class="btn btn-ghost btn-circle absolute right-2 top-2"
        onclick={close}
        aria-label="Close"
        title="Close"
      >
        <X class="w-5 h-5" />
      </button>
      <h3 class="font-bold text-lg mb-4 flex items-center gap-2">
        {#if $profileModal.kind === 'ethernet'}
          <Cable class="w-5 h-5" />
        {:else if $profileModal.kind === 'modem'}
          <Smartphone class="w-5 h-5" />
        {:else}
          <Wifi class="w-5 h-5" />
        {/if}
        Connect {$profileModal.iface}
      </h3>

      {#if $profileModal.profiles.length === 0}
        <div class="text-center py-6">
          {#if $profileModal.kind === 'ethernet'}
            <Cable class="w-14 h-14 mx-auto mb-3 text-base-content/20" />
          {:else if $profileModal.kind === 'modem'}
            <Smartphone class="w-14 h-14 mx-auto mb-3 text-base-content/20" />
          {:else}
            <Wifi class="w-14 h-14 mx-auto mb-3 text-base-content/20" />
          {/if}
          <p class="font-medium">No saved profiles for this interface</p>
          {#if $profileModal.kind === 'wifi'}
            <p class="text-sm text-base-content/60 mt-1">
              Scan for networks and connect in the Wi-Fi section to create a profile.
            </p>
          {:else}
            <p class="text-sm text-base-content/60 mt-1">
              Create a connection profile in the Connections section.
            </p>
          {/if}
          <button type="button" class="btn btn-primary mt-5" onclick={handleManage}>Manage</button>
        </div>
      {:else}
        <div class="space-y-2 max-h-72 overflow-y-auto pr-1">
          {#each $profileModal.profiles as conn}
            <button
              type="button"
              class="w-full flex items-center gap-3 p-3 rounded-lg border border-base-300 text-left hover:bg-base-200/60 transition-colors"
              onclick={() => handleSelect(conn)}
              title={`Connect to profile "${conn.id}"`}
            >
              <div class="p-2 bg-primary/10 rounded-lg shrink-0">
                <svelte:component this={typeIcon(conn)} class="w-4 h-4 text-primary" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-medium truncate">{conn.id}</span>
                  {#if conn.active}
                    <span class="badge badge-success badge-sm shrink-0">Active</span>
                  {/if}
                  {#if conn.is_wifi && conn.ssid}
                    <span class="badge badge-ghost badge-sm shrink-0">"{conn.ssid}"</span>
                  {/if}
                </div>
                <p class="text-xs text-base-content/60 truncate mt-0.5">
                  {conn.interface || 'Any interface'} • {conn.ipv4_method} / {conn.ipv6_method}
                </p>
              </div>
              <span class="badge badge-primary badge-sm shrink-0">Connect</span>
            </button>
          {/each}
        </div>
        <div class="modal-action mt-4">
          <button type="button" class="btn btn-ghost" onclick={close}>Cancel</button>
          <button type="button" class="btn" onclick={handleManage}>Manage</button>
        </div>
      {/if}
    </div>
    <button type="button" class="modal-backdrop" onclick={close} aria-hidden="true"></button>
  </div>
{/if}