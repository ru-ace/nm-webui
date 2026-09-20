<script lang="ts">
  import { onMount } from 'svelte';
  import { CheckCircle, AlertTriangle, XCircle, WifiOff, Wifi, Signal, Shield, Loader2, ChevronDown, ChevronUp, RefreshCw, Smartphone } from 'lucide-svelte';
  import {
    systemStatus,
    devices,
    wifiStatusMap,
    connectivityStatus,
    isOnline,
    loading,
    refreshExternalIP,
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
  $: modemDevices = $devices.filter(d => !!d.modem && d.managed);
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

  <div class="grid grid-cols-1 gap-4">
    <div class="card bg-base-100 shadow-sm border border-base-300">
      <div class="card-body">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-sm text-base-content/60">External IP</p>
            {#if $systemStatus?.external_ip}
              <a
                class="font-mono text-lg font-medium link link-primary"
                href={`https://www.iplocation.net/ip-lookup?query=${encodeURIComponent($systemStatus.external_ip)}`}
                target="_blank"
                rel="noreferrer"
              >{$systemStatus.external_ip}</a>
            {:else}
              <p class="font-mono text-lg font-medium">Unavailable</p>
            {/if}
            {#if $systemStatus?.external_ip_city || $systemStatus?.external_ip_country}
              <p class="text-sm text-base-content/70">
                {#if $systemStatus?.external_ip_city}{$systemStatus.external_ip_city}{/if}{#if $systemStatus?.external_ip_city && $systemStatus?.external_ip_country}, {/if}{$systemStatus?.external_ip_country || ''}
              </p>
            {/if}
            <p class="text-xs text-base-content/50">
              {#if $systemStatus?.external_ip_status === 'cached'}
                Cached result
              {:else if $systemStatus?.external_ip_status === 'fresh'}
                Verified via HTTPS
              {:else}
                {$isOnline ? 'HTTPS check failed' : 'Internet unavailable'}
              {/if}
            </p>
            {#if $systemStatus?.external_ip_isp || $systemStatus?.external_ip_asn || $systemStatus?.external_ip_timezone}
              <dl class="grid grid-cols-1 sm:grid-cols-3 gap-x-6 gap-y-2 mt-3 text-sm">
                {#if $systemStatus?.external_ip_isp}
                  <div class="min-w-0">
                    <dt class="text-base-content/60">ISP</dt>
                    <dd class="font-medium truncate" title={$systemStatus.external_ip_isp}>{$systemStatus.external_ip_isp}</dd>
                  </div>
                {/if}
                {#if $systemStatus?.external_ip_asn}
                  <div class="min-w-0">
                    <dt class="text-base-content/60">ASN</dt>
                    <dd class="font-mono text-xs truncate" title={$systemStatus.external_ip_asn}>{$systemStatus.external_ip_asn}</dd>
                  </div>
                {/if}
                {#if $systemStatus?.external_ip_timezone}
                  <div class="min-w-0">
                    <dt class="text-base-content/60">Timezone</dt>
                    <dd class="font-medium truncate" title={$systemStatus.external_ip_timezone}>{$systemStatus.external_ip_timezone}</dd>
                  </div>
                {/if}
              </dl>
            {/if}
          </div>
          <div class="relative h-12 w-12 shrink-0">
            <Shield class="h-12 w-12 text-base-content/20" />
            <button
              type="button"
              class="absolute inset-0 flex items-center justify-center rounded-full text-base-content/60 transition-colors hover:bg-base-200 hover:text-primary disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-base-content/60"
              onclick={refreshExternalIP}
              disabled={!$isOnline || $loading['external-ip']}
              title={$isOnline ? 'Refresh external IP' : 'Internet is unavailable'}
              aria-label="Refresh external IP"
            >
              <RefreshCw class="h-5 w-5 {$loading['external-ip'] ? 'animate-spin' : ''}" />
            </button>
          </div>
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

  {#if modemDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Smartphone class="w-5 h-5 text-primary" />
        Mobile Broadband
      </h2>
      <div class="grid gap-4 md:grid-cols-2">
        {#each modemDevices as device}
          <div class="card bg-base-100 shadow-sm border border-base-300 card-hover">
            <div class="card-body">
              <div class="flex items-start justify-between mb-4">
                <div>
                  <h3 class="font-semibold">{device.interface}</h3>
                  <p class="text-sm text-base-content/60">{device.modem?.manufacturer || ''} {device.modem?.model || ''}</p>
                </div>
                <span class="badge {getStateBadge(device.state).class}">{getStateBadge(device.state).label}</span>
              </div>

              <div class="space-y-2 mb-4 p-3 rounded-lg {device.state === 100 ? 'bg-success/10' : 'bg-base-200/50'}">
                <div class="flex items-center justify-between flex-wrap gap-1">
                  <div class="flex items-center gap-2">
                    <span class="font-medium">{device.modem?.operator_name || '—'}</span>
                    {#if device.modem?.access_tech_name}
                      <span class="badge badge-primary badge-sm">{device.modem.access_tech_name}</span>
                    {/if}
                  </div>
                  <div class="flex items-center gap-2">
                    {#if device.modem?.signal}
                      <span class="badge badge-ghost badge-sm">Signal {device.modem.signal.percent}%</span>
                    {/if}
                    {#if device.state === 100}
                      <span class="text-xs text-success flex items-center gap-1"><Signal class="w-4 h-4" /> Connected</span>
                    {:else if device.state === 30}
                      <span class="text-xs text-base-content/50">Not connected</span>
                    {:else}
                      <span class="text-xs text-base-content/50">Modem: {device.modem?.state_name || 'preparing'}</span>
                    {/if}
                  </div>
                </div>
                {#if device.state === 100 && device.ipv4?.addresses?.length}
                  <p class="text-xs font-mono text-base-content/70">
                    {device.ipv4.addresses.map((a: { address: string; prefix: number }) => `${a.address}/${a.prefix}`).join(', ')}
                  </p>
                {/if}
                <div class="grid grid-cols-2 gap-2 text-xs">
                  <div><span class="text-base-content/60">APN</span><br /><span class="font-mono">{device.modem?.apn || '—'}</span></div>
                  <div><span class="text-base-content/60">SIM</span><br /><span class="font-mono">{device.modem?.sim?.operator_name || device.modem?.sim?.iccid || '—'}</span></div>
                </div>
              </div>

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
                <button type="button" class="btn btn-ghost" onclick={() => navigate('/devices')}>Manage</button>
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
