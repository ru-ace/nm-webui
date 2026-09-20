<script lang="ts">
  import { onMount } from 'svelte';
  import { CheckCircle, AlertTriangle, XCircle, Globe, WifiOff, Wifi, Shield, Zap, Loader2, ChevronDown, ChevronUp } from 'lucide-svelte';
  import {
    systemStatus,
    devices,
    wifiStatusMap,
    connectivityStatus,
    isOnline,
    loading,
    loadWifiStatus,
    disconnectDevice,
    connectDevice
  } from '$lib/stores/app';
  import { navigate } from '$lib/stores/router';
  import { derived } from 'svelte/store';

  onMount(() => {
    $devices.filter(d => d.wireless).forEach(d => loadWifiStatus(d.interface));
  });

  $: wifiDevices = $devices.filter(d => d.wireless && d.managed);
  $: primaryWifi = wifiDevices.find(d => d.state === 100);
  $: primaryWifiStatus = primaryWifi ? $wifiStatusMap[primaryWifi.interface] : null;
  $: loadingMap = $loading;

  function getConnectivityClass(status: string) {
    if (status === 'online') return 'badge-success';
    if (status === 'limited') return 'badge-warning';
    return 'badge-error';
  }

  function getStateBadge(state: number) {
    const states: Record<number, { label: string; class: string }> = {
      100: { label: 'Connected', class: 'badge-success' },
      70: { label: 'Getting IP', class: 'badge-warning' },
      50: { label: 'Configuring', class: 'badge-warning' },
      30: { label: 'Disconnected', class: 'badge-ghost' },
      20: { label: 'Unavailable', class: 'badge-error' },
      10: { label: 'Unmanaged', class: 'badge-neutral' }
    };
    return states[state] || { label: 'Unknown', class: 'badge-ghost' };
  }
</script>

<div class="space-y-6 max-w-4xl mx-auto">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold">Dashboard</h1>
    <div class="flex items-center gap-2">
      <span class="badge {getConnectivityClass($connectivityStatus)} gap-1">
        {#if $connectivityStatus === 'online'}
          <CheckCircle class="w-4 h-4" />
        {:else if $connectivityStatus === 'limited'}
          <AlertTriangle class="w-4 h-4" />
        {:else}
          <XCircle class="w-4 h-4" />
        {/if}
        {$connectivityStatus}
      </span>
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <div class="card bg-base-100 shadow-sm border border-base-300">
      <div class="card-body">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-base-content/60">Internet</p>
            <p class="text-3xl font-bold flex items-center gap-2">
              {#if $isOnline}
                <CheckCircle class="w-8 h-8 text-success" />
                <span>Online</span>
              {:else if $connectivityStatus === 'limited'}
                <AlertTriangle class="w-8 h-8 text-warning" />
                <span>Limited</span>
              {:else}
                <XCircle class="w-8 h-8 text-error" />
                <span>Offline</span>
              {/if}
            </p>
          </div>
          <Globe class="w-12 h-12 text-base-content/20" />
        </div>
      </div>
    </div>

    <div class="card bg-base-100 shadow-sm border border-base-300">
      <div class="card-body">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-base-content/60">External IP</p>
            <p class="font-mono text-lg font-medium">{$systemStatus?.external_ip || '—'}</p>
          </div>
          <Shield class="w-12 h-12 text-base-content/20" />
        </div>
      </div>
    </div>

    <div class="card bg-base-100 shadow-sm border border-base-300">
      <div class="card-body">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-base-content/60">Gateway</p>
            <p class="font-mono text-lg font-medium">{$systemStatus?.primary_gateway || '—'}</p>
          </div>
          <Zap class="w-12 h-12 text-base-content/20" />
        </div>
      </div>
    </div>
  </div>

  {#if wifiDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3">Wi-Fi Interfaces</h2>
      <div class="grid gap-4 md:grid-cols-2">
        {#each wifiDevices as device}
          <div class="card bg-base-100 shadow-sm border border-base-300 card-hover">
            <div class="card-body">
              <div class="flex items-start justify-between mb-4">
                <div>
                  <h3 class="font-semibold">{device.interface}</h3>
                  <p class="text-sm text-base-content/60">{device.mac}</p>
                </div>
                <span class="badge {getStateBadge(device.state).class}">{getStateBadge(device.state).label}</span>
              </div>

              {#if primaryWifiStatus?.connected && device.interface === primaryWifi?.interface}
                <div class="space-y-2 mb-4 p-3 bg-success/10 rounded-lg">
                  <div class="flex items-center justify-between">
                    <div>
                      <p class="font-medium">{primaryWifiStatus.ssid}</p>
                      <p class="text-sm text-base-content/60">{primaryWifiStatus.security?.toUpperCase() || 'Open'} • {primaryWifiStatus.signal}% • {primaryWifiStatus.band || '?'} GHz</p>
                    </div>
                    <Wifi class="w-8 h-8 text-success" />
                  </div>
                </div>
              {:else if device.state === 30}
                <div class="text-center py-4 text-base-content/50">
                  <WifiOff class="w-12 h-12 mx-auto mb-2 opacity-50" />
                  <p>Not connected</p>
                </div>
              {:else if device.state > 30 && device.state < 100}
                <div class="flex items-center justify-center gap-2 py-4">
                  <Loader2 class="w-6 h-6 animate-spin text-primary" />
                  <span>Connecting...</span>
                </div>
              {/if}

              <div class="flex gap-2">
                {#if device.state === 100}
                  <button
                    type="button"
                    class="btn btn-error flex-1"
                    onclick={() => disconnectDevice(device.interface)}
                    disabled={loadingMap[`disconnect-${device.interface}`]}
                  >
                    Disconnect
                  </button>
                {:else if device.state === 30}
                  <button
                    type="button"
                    class="btn btn-primary flex-1"
                    onclick={() => connectDevice(device.interface)}
                    disabled={loadingMap[`connect-${device.interface}`]}
                  >
                    Connect
                  </button>
                {:else}
                  <button type="button" class="btn btn-ghost flex-1" disabled>Please wait...</button>
                {/if}
                <button type="button" class="btn btn-ghost" onclick={() => navigate(`/wifi?iface=${device.interface}`)}>Manage</button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}

  <section>
    <h2 class="text-lg font-semibold mb-3">System Info</h2>
    <div class="card bg-base-100 shadow-sm border border-base-300">
      <div class="card-body">
        <dl class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div><dt class="text-base-content/60">Hostname</dt><dd class="font-medium">{$systemStatus?.hostname || '—'}</dd></div>
          <div><dt class="text-base-content/60">NM Version</dt><dd class="font-medium">{$systemStatus?.networkmanager_version || '—'}</dd></div>
          <div><dt class="text-base-content/60">State</dt><dd class="font-medium">{$systemStatus?.state_name || '—'}</dd></div>
          <div><dt class="text-base-content/60">Networking</dt><dd class="font-medium">{$systemStatus?.networking_enabled ? 'Enabled' : 'Disabled'}</dd></div>
        </dl>
      </div>
    </div>
  </section>
</div>