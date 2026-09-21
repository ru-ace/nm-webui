<script lang="ts">
  import { onMount } from 'svelte';
  import { Plus, Trash2, ToggleLeft, ToggleRight, Loader2, Wifi, Cable, Database, Copy, ChevronDown, ChevronUp, Settings, Smartphone } from '@lucide/svelte';
  import { connections, loading, loadConnections, forgetConnection, toggleAutoconnect, activateConnection, deactivateConnection } from '$lib/stores/app';
  import ConnectionModal from './ConnectionModal.svelte';
  import ConnectionRow from './ConnectionRow.svelte';

  let showModal = false;
  let editingConnection: any = null;

  onMount(() => loadConnections());

  $: wifiConnections = $connections.filter(c => c.is_wifi);
  $: ethernetConnections = $connections.filter(c => !c.is_wifi && c.type_name === 'ethernet');
  $: modemConnections = $connections.filter(c => c.is_modem);
  $: otherConnections = $connections.filter(c => !c.is_wifi && c.type_name !== 'ethernet' && !c.is_modem);
  $: loadingMap = $loading;

  function handleSubmit(event: CustomEvent<{ data: any; editing: boolean }>) {
    const { data, editing } = event.detail;
    if (editing) {
      // Update handled in modal
    } else {
      // Create handled in modal
    }
    showModal = false;
    editingConnection = null;
  }
</script>

<div class="space-y-6 max-w-4xl mx-auto">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Settings class="w-7 h-7 text-primary" />
      <h1 class="text-2xl font-bold">Connection Profiles</h1>
    </div>
    <button class="btn btn-primary gap-2" onclick={() => { editingConnection = null; showModal = true; }}>
      <Plus class="w-4 h-4" />
      New Profile
    </button>
  </div>

  <ConnectionModal bind:show={showModal} bind:editing={editingConnection} on:submit={handleSubmit} />

  {#if $connections.length === 0 && !$loading.connections}
    <div class="text-center py-12">
      <Database class="w-16 h-16 mx-auto mb-4 text-base-content/20" />
      <p class="text-base-content/60">No connection profiles</p>
      <button class="btn btn-primary mt-4" onclick={() => { editingConnection = null; showModal = true; }}>Create First Profile</button>
    </div>
  {/if}

  {#each [
    { title: 'Wi-Fi', items: wifiConnections, icon: Wifi },
    { title: 'Ethernet', items: ethernetConnections, icon: Cable },
    { title: 'Mobile Broadband', items: modemConnections, icon: Smartphone },
    { title: 'Other', items: otherConnections, icon: Database }
  ] as group}
    {#if group.items.length > 0}
      <section class="space-y-3">
        <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
          <svelte:component this={group.icon} class="w-5 h-5 text-primary" />
          {group.title}
        </h2>
        <div class="space-y-2">
          {#each group.items as conn}
            <ConnectionRow {conn} {loadingMap} onEdit={(c) => { editingConnection = c; showModal = true; }} />
          {/each}
        </div>
      </section>
    {/if}
  {/each}
</div>