<script lang="ts">
  import { onMount } from 'svelte';
  import { Server, Cable, Wifi, Database, Cpu, Loader2, ChevronDown, ChevronUp, Settings, WifiOff, Link2, Unlink2 } from 'lucide-svelte';
  import { devices, loading, loadDevices, activateConnection, deactivateConnection } from '$lib/stores/app';
  import DeviceCard from './DeviceCard.svelte';

  onMount(() => loadDevices());

  $: ethernetDevices = $devices.filter(d => d.type_name === 'ethernet');
  $: wifiDevices = $devices.filter(d => d.wireless);
  $: otherDevices = $devices.filter(d => d.type_name !== 'ethernet' && !d.wireless);
  $: loadingMap = $loading;
</script>

<div class="space-y-6 max-w-6xl mx-auto">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold">Network Devices</h1>
    <button class="btn btn-primary gap-2" onclick={loadDevices} disabled={$loading.devices}>
      <div class="w-4 h-4" class:animate-spin={$loading.devices}><Loader2 class="w-4 h-4" /></div>
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