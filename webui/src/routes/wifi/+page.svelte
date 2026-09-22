<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Wifi, Loader2, Lock, Unlock, Shield, SignalHigh, SignalLow, RefreshCw } from '@lucide/svelte';
  import {
    devices,
    wifiNetworks,
    loading,
    triggerScan,
    loadWifiNetworks,
    connectWifi,
    loadWifiStatus
  } from '$lib/stores/app';
  import { passwordModal, showPasswordModal } from '$lib/stores/modals';
  import { navigate } from '$lib/stores/router';
  import { headerAction } from '$lib/stores/header';

  let selectedIface = '';
  let showPasswords = false;
  let scanning = false;
  let autoScanTimer: ReturnType<typeof setInterval> | undefined;

  $: wifiDevices = $devices.filter(d => d.wireless && d.managed);
  $: selectedDeviceObj = wifiDevices.find(d => d.interface === selectedIface);
  $: loadingMap = $loading;

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    selectedIface = params.get('iface') || '';
    if (selectedIface) loadWifiNetworks(selectedIface);
    // NetworkManager prunes access points that stop emitting beacons, so the
    // network list collapses to the current network after a while. Re-scan
    // quietly every 60 seconds to keep the list populated.
    autoScanTimer = setInterval(() => {
      void handleScan().catch(() => {});
    }, 60_000);
  });

  onDestroy(() => {
    if (autoScanTimer) clearInterval(autoScanTimer);
    headerAction.set(null);
  });

  $: if (!selectedIface && wifiDevices.length > 0) {
    selectedIface = wifiDevices[0].interface;
    navigate(`/wifi?iface=${selectedIface}`);
    loadWifiNetworks(selectedIface);
  }

  async function handleScan() {
    if (!selectedIface) return;
    scanning = true;
    try {
      await triggerScan(selectedIface);
    } finally {
      scanning = false;
    }
  }

  // Mobile header action: icon-only Scan button, kept in sync with the local
  // scanning state and the network loading flag.
  $: headerAction.set({
    label: 'Scan for networks',
    icon: RefreshCw,
    onClick: () => { void handleScan(); },
    disabled: scanning || !!loadingMap[`wifi-${selectedIface}`],
    loading: scanning
  });

  async function handleConnect(network: any) {
    if (!selectedIface) return;
    if (network.saved) {
      await connectWifi(selectedIface, network.ssid);
    } else if (network.security !== 'open') {
      const password = await showPasswordModal(network.ssid);
      if (password === null) return;
      await connectWifi(selectedIface, network.ssid, password);
    } else {
      await connectWifi(selectedIface, network.ssid);
    }
  }

  function getSecurityComponent(security: string) {
    if (security === 'open') return Unlock;
    if (security === 'enterprise') return Shield;
    if (security === 'wpa3') return Lock;
    return Lock;
  }

  function getSecurityLabel(security: string) {
    return security === 'open' ? 'Open network' : `${security.toUpperCase()} secured network`;
  }

  function getSignalComponent(signal: number) {
    if (signal >= 75) return SignalHigh;
    if (signal >= 50) return SignalHigh;
    if (signal >= 25) return SignalLow;
    return SignalLow;
  }

  function getSignalClass(signal: number) {
    if (signal >= 75) return '';
    if (signal >= 50) return 'opacity-75';
    if (signal >= 25) return '';
    return 'opacity-50';
  }

  // Short band digit ("2" for 2.4/2, "5" for 5, "6" for 6).
  function getBandLabel(band?: string) {
    if (!band) return '?';
    const n = Number.parseFloat(band);
    return Number.isNaN(n) ? '?' : String(Math.trunc(n));
  }
</script>

<div class="space-y-6 max-w-4xl mx-auto">
  <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
    <div class="hidden lg:flex items-center gap-2">
      <Wifi class="w-7 h-7 text-primary" />
      <h1 class="text-2xl font-bold">Wi-Fi</h1>
    </div>
    <div class="flex items-center gap-2">
      <select
        bind:value={selectedIface}
        onchange={() => {
          navigate(`/wifi?iface=${selectedIface}`);
          loadWifiNetworks(selectedIface);
        }}
        class="select select-bordered w-full sm:w-48"
      >
        {#each wifiDevices as d}
          <option value={d.interface}>{d.interface} ({d.mac})</option>
        {/each}
      </select>
      <button
        class="btn btn-primary gap-2 hidden lg:inline-flex"
        onclick={handleScan}
        disabled={scanning || loadingMap[`wifi-${selectedIface}`]}
      >
        <div class="w-4 h-4" class:animate-spin={scanning}><RefreshCw class="w-4 h-4" /></div>
        Scan
      </button>
    </div>
  </div>

  {#if selectedDeviceObj}
    <div class="card bg-base-100 shadow-sm border border-base-300 mb-4">
      <div class="card-body">
        <div class="flex items-center justify-between flex-wrap gap-2">
          <div class="flex items-center gap-3">
            <Wifi class="w-6 h-6 text-primary" />
            <div>
              <p class="font-medium">{selectedDeviceObj.interface}</p>
              <p class="text-sm text-base-content/60">
                {selectedDeviceObj.mac} • MTU {selectedDeviceObj.mtu}
              </p>
            </div>
          </div>
          <span class="badge badge-lg badge-success">Ready</span>
        </div>
      </div>
    </div>
  {/if}

  <div class="card bg-base-100 shadow-sm border border-base-300">
    <div class="card-body p-0">
      {#if loadingMap[`wifi-${selectedIface}`] && $wifiNetworks.length === 0}
        <div class="flex items-center justify-center py-12">
          <Loader2 class="w-8 h-8 animate-spin text-primary" />
        </div>
      {:else if $wifiNetworks.length === 0}
        <div class="text-center py-12">
          <Wifi class="w-16 h-16 mx-auto mb-4 text-base-content/20" />
          <p class="text-base-content/60">No networks found</p>
          <button class="btn btn-primary mt-4" onclick={handleScan}>Scan for Networks</button>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table table-fixed w-full">
            <thead>
              <tr class="bg-base-200">
                <th class="w-[12%] px-2 py-3"><span class="sr-only">Signal</span></th>
                <th class="w-[45%] px-2 py-3">Network</th>
                <th class="w-[22%] px-2 py-3 text-center">Level</th>
                <th class="w-[21%] px-2 py-3 text-right">Action</th>
              </tr>
            </thead>
            <tbody>
              {#each $wifiNetworks as network (network.ssid)}
                <tr class="hover:bg-base-100">
                  <td class="px-2 py-3">
                    <div class="relative inline-flex h-9 w-9 items-center justify-center">
                      <Wifi class="h-6 w-6 text-base-content/60" />
                      {#if network.security !== 'open'}
                        <span
                          class="absolute bottom-0 left-0 rounded bg-base-100 leading-none"
                          title={getSecurityLabel(network.security)}
                          aria-label={getSecurityLabel(network.security)}
                        >
                          <svelte:component this={getSecurityComponent(network.security)} class="h-3 w-3 {network.security === 'wpa3' ? 'text-success' : 'text-warning'}" />
                        </span>
                      {/if}
                      <span
                        class="absolute -bottom-[3px] right-0 flex h-4 w-3 items-center justify-center rounded bg-base-100 font-mono text-sm font-semibold leading-none text-base-content/70"
                        title={network.band ? `${network.band} GHz band` : 'Unknown band'}
                      >
                        {getBandLabel(network.band)}
                      </span>
                    </div>
                  </td>
                  <td class="min-w-0 px-2 py-3">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="truncate font-medium">{network.ssid || '(hidden)'}</span>
                      {#if network.saved}
                        <span class="badge badge-ghost badge-xs shrink-0">Saved</span>
                      {/if}
                      {#if network.bssids && network.bssids.length > 1}
                        <span class="badge badge-ghost badge-xs shrink-0">{network.bssids.length} APs</span>
                      {/if}
                    </div>
                  </td>
                  <td class="px-2 py-3 whitespace-nowrap text-center">
                    <div class="flex items-center justify-center gap-1">
                      <svelte:component this={getSignalComponent(network.signal)} class="h-5 w-5 {getSignalClass(network.signal)}" />
                      <span class="font-mono">{network.signal}%</span>
                    </div>
                  </td>
                  <td class="px-2 py-3 text-right whitespace-nowrap">
                    <button
                      class="btn btn-primary btn-sm"
                      onclick={() => handleConnect(network)}
                      disabled={loadingMap[`connect-${selectedIface}`]}
                    >
                      Join
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  </div>
</div>
