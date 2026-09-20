<script lang="ts">
  import { onMount } from 'svelte';
  import { Wifi, Search, Loader2, Lock, Unlock, SignalHigh, SignalLow, RefreshCw, ChevronDown, ChevronUp } from 'lucide-svelte';
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

  let selectedIface = '';
  let showPasswords = false;
  let scanning = false;

  $: wifiDevices = $devices.filter(d => d.wireless && d.managed);
  $: selectedDeviceObj = wifiDevices.find(d => d.interface === selectedIface);
  $: loadingMap = $loading;

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    selectedIface = params.get('iface') || '';
    if (!selectedIface && wifiDevices.length > 0) {
      selectedIface = wifiDevices[0].interface;
      navigate(`/wifi?iface=${selectedIface}`);
    }
    if (selectedIface) loadWifiNetworks(selectedIface);
  });

  async function handleScan() {
    if (!selectedIface) return;
    scanning = true;
    await triggerScan(selectedIface);
    scanning = false;
  }

  async function handleConnect(network: any) {
    if (!selectedIface) return;
    if (network.security !== 'open') {
      const password = await showPasswordModal(network.ssid);
      if (password === null) return;
      await connectWifi(selectedIface, network.ssid, password);
    } else {
      await connectWifi(selectedIface, network.ssid);
    }
  }

  function getSecurityComponent(security: string) {
    if (security === 'open') return Unlock;
    if (security === 'wpa3') return Lock;
    return Lock;
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
</script>

<div class="space-y-6 max-w-4xl mx-auto">
  <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
    <h1 class="text-2xl font-bold">Wi-Fi Networks</h1>
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
        class="btn btn-primary gap-2"
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
          <table class="table w-full">
            <thead>
              <tr class="bg-base-200">
                <th class="px-4 py-3">Network</th>
                <th class="px-4 py-3 text-center">Signal</th>
                <th class="px-4 py-3 text-center">Security</th>
                <th class="px-4 py-3 text-center">Band</th>
                <th class="px-4 py-3 text-center">Channel</th>
                <th class="px-4 py-3 text-right">Action</th>
              </tr>
            </thead>
            <tbody>
              {#each $wifiNetworks as network (network.ssid)}
                <tr class="hover:bg-base-100">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-3">
                      {#if network.bssids && network.bssids.length > 1}
                        <details class="group">
                          <summary class="flex items-center gap-2 cursor-pointer list-none">
                            <Wifi class="w-5 h-5 text-base-content/60" />
                            <span class="font-medium">{network.ssid}</span>
                            <span class="badge badge-ghost badge-xs">{network.bssids.length} APs</span>
                          </summary>
                          <ul class="pl-8 pt-2 space-y-1 border-l border-base-300 ml-2">
                            {#each network.bssids as bssid}
                              <li class="text-sm text-base-content/70 flex items-center gap-2">
                                <span class="font-mono text-xs">{bssid.bssid}</span>
                                <span>{bssid.signal}%</span>
                                <span class="badge badge-ghost badge-xs">{bssid.security}</span>
                              </li>
                            {/each}
                          </ul>
                        </details>
                      {:else}
                        <Wifi class="w-5 h-5 text-base-content/60" />
                        <span class="font-medium">{network.ssid || '(hidden)'}</span>
                      {/if}
                    </div>
                  </td>
                  <td class="px-4 py-3 text-center">
                    <div class="flex items-center justify-center gap-2">
                      <svelte:component this={getSignalComponent(network.signal)} class="w-5 h-5 {getSignalClass(network.signal)}" />
                      <span class="font-mono">{network.signal}%</span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-center">
                    <div class="flex items-center justify-center gap-1">
                      <svelte:component this={getSecurityComponent(network.security)} class="w-4 h-4 {network.security === 'wpa3' ? 'text-success' : ''}" />
                      <span class="badge badge-ghost badge-sm">{network.security.toUpperCase()}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-center">
                    <span class="badge badge-ghost">{network.band || '?'}</span>
                  </td>
                  <td class="px-4 py-3 text-center">
                    <span class="font-mono">{network.channel || '—'}</span>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      class="btn btn-primary btn-sm"
                      onclick={() => handleConnect(network)}
                      disabled={loadingMap[`connect-${selectedIface}`]}
                    >
                      {network.security === 'open' ? 'Connect' : 'Join'}
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