<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Server, Cable, Wifi, Database, Cpu, Loader2, RefreshCw, ChevronDown, ChevronUp, Settings, WifiOff, Link2, Unlink2, Smartphone } from '@lucide/svelte';
  import { devices, loading, loadDevices, activateConnection, deactivateConnection } from '$lib/stores/app';
  import { headerAction } from '$lib/stores/header';
  import DeviceCard from './DeviceCard.svelte';

  onMount(() => loadDevices());
  onDestroy(() => headerAction.set(null));

  // Mobile header action: icon-only Refresh button.
  $: headerAction.set({
    label: 'Refresh devices',
    icon: RefreshCw,
    onClick: () => { void loadDevices(); },
    disabled: $loading.devices,
    loading: $loading.devices
  });

  $: ethernetDevices = $devices.filter(d => d.type_name === 'ethernet');
  $: wifiDevices = $devices.filter(d => d.wireless);
  $: modemDevices = $devices.filter(d => !!d.modem);
  $: otherDevices = $devices.filter(d => d.type_name !== 'ethernet' && !d.wireless && !d.modem);
  $: loadingMap = $loading;
</script>

<div class="space-y-6 max-w-6xl mx-auto">
  <div class="hidden lg:flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Server class="w-7 h-7 text-primary" />
      <h1 class="text-2xl font-bold">Devices</h1>
    </div>
    <button class="btn btn-primary gap-2" onclick={loadDevices} disabled={$loading.devices}>
      {#if $loading.devices}
        <Loader2 class="w-4 h-4 animate-spin" />
      {:else}
        <RefreshCw class="w-4 h-4" />
      {/if}
      Refresh
    </button>
  </div>

  {#if ethernetDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Cable class="w-5 h-5 text-primary" />
        Ethernet
      </h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {#each ethernetDevices as device}
          <DeviceCard {device} {loadingMap} />
        {/each}
      </div>
    </section>
  {/if}

  {#if wifiDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Wifi class="w-5 h-5 text-primary" />
        Wi-Fi
      </h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {#each wifiDevices as device}
          <DeviceCard {device} {loadingMap} />
        {/each}
      </div>
    </section>
  {/if}

  {#if modemDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Smartphone class="w-5 h-5 text-primary" />
        Mobile Broadband
      </h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {#each modemDevices as device}
          <DeviceCard {device} {loadingMap} />
        {/each}
      </div>
    </section>
  {/if}

  {#if otherDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Server class="w-5 h-5 text-primary" />
        Other
      </h2>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {#each otherDevices as device}
          <DeviceCard {device} {loadingMap} />
        {/each}
      </div>
    </section>
  {/if}

  {#if $devices.length === 0 && !$loading.devices}
    <div class="text-center py-12">
      <Server class="w-16 h-16 mx-auto mb-4 text-base-content/20" />
      <p class="text-base-content/60">No network devices found</p>
    </div>
  {/if}
</div>