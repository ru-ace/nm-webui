<script lang="ts">
  import { onMount } from 'svelte';
  import { CheckCircle, AlertTriangle, XCircle, WifiOff, Wifi, Signal, Shield, Loader2, ChevronDown, ChevronUp, RefreshCw, Smartphone, EthernetPort, ShieldAlert, Monitor } from '@lucide/svelte';
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
    activateConnection,
    showToast,
    captivePortal,
    recheckCaptivePortal
  } from '$lib/stores/app';
  import { showProfileModal } from '$lib/stores/modals';
  import { api } from '$lib/api/client';
  import { navigate } from '$lib/stores/router';
  import { derived } from 'svelte/store';

  onMount(() => {
    $devices.filter(d => d.wireless).forEach(d => loadWifiStatus(d.interface));
  });

  let connectingIface: string | null = null;
  $: wifiDevices = $devices.filter(d => d.wireless && d.managed);
  $: ethernetDevices = $devices.filter(d => d.type_name === 'ethernet' && d.managed && d.state >= 30);
  $: modemDevices = $devices.filter(d => !!d.modem && d.managed);
  $: primaryWifi = wifiDevices.find(d => d.state === 100);
  $: primaryWifiStatus = primaryWifi ? $wifiStatusMap[primaryWifi.interface] : null;
  $: loadingMap = $loading;

  function getConnectivityClass(status: string) {
    if (status === 'online') return 'badge-success';
    if (status === 'limited' || status === 'portal') return 'badge-warning';
    return 'badge-error';
  }

  function getStateBadge(state: number) {
    const states: Record<number, { label: string; class: string }> = {
      100: { label: 'Connected', class: 'badge-success' },
      110: { label: 'Disconnecting', class: 'badge-warning' },
      70: { label: 'Getting IP', class: 'badge-warning' },
      50: { label: 'Configuring', class: 'badge-warning' },
      30: { label: 'Disconnected', class: 'badge-ghost' },
      20: { label: 'Unavailable', class: 'badge-error' },
      10: { label: 'Unmanaged', class: 'badge-neutral' }
    };
    return states[state] || { label: 'Unknown', class: 'badge-ghost' };
  }

  // Dashboard card "Connect": ask NetworkManager which saved profiles are
  // applicable to this interface and either activate directly (single
  // profile), or let the user pick one. With no applicable profiles the modal
  // falls back to a Manage route (Wi-Fi section for wireless cards, the
  // Connections page for wired and modem ones, where profiles are created).
  async function handleCardConnect(
    device: { interface: string; type_name?: string },
    manage: string,
    kind: 'wifi' | 'ethernet'
  ) {
    const iface = device.interface;
    connectingIface = iface;
    try {
      const { connections: profiles } = await api.devices.connections(iface);
      if (profiles.length === 0) {
        await showProfileModal(iface, profiles, { manage, kind });
      } else if (profiles.length === 1) {
        await activateConnection(profiles[0].uuid);
      } else {
        const selected = await showProfileModal(iface, profiles, { manage, kind });
        if (selected) {
          await activateConnection(selected.uuid);
        }
      }
    } catch (e: any) {
      console.error('Failed to load device profiles:', e);
      showToast(`Failed to load profiles: ${e?.message || e}`, 'error');
    } finally {
      connectingIface = null;
    }
  }
</script>

<div class="space-y-6 w-full max-w-4xl mx-auto">
  <div class="hidden lg:flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Monitor class="w-7 h-7 text-primary" />
      <h1 class="text-2xl font-bold">Dashboard</h1>
    </div>
    <div class="flex items-center gap-2">
      <span class="badge {getConnectivityClass($connectivityStatus)} gap-1">
        {#if $connectivityStatus === 'online'}
          <CheckCircle class="w-4 h-4" />
        {:else if $connectivityStatus === 'limited' || $connectivityStatus === 'portal'}
          <AlertTriangle class="w-4 h-4" />
        {:else}
          <XCircle class="w-4 h-4" />
        {/if}
        {$connectivityStatus}
      </span>
    </div>
  </div>

  {#if $connectivityStatus === 'portal' || $captivePortal?.state === 'portal'}
    <div class="alert alert-warning shadow-sm">
      <ShieldAlert class="w-6 h-6 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold">Captive portal detected</p>
        <p class="text-sm text-warning-content/70">Sign in to the network on the host to unlock internet access for clients.</p>
      </div>
      <div class="flex gap-2 shrink-0">
        <button
          type="button"
          class="btn btn-sm"
          onclick={() => {
            const portalUrl = $captivePortal?.portal_url;
            navigate(portalUrl ? `/portal?url=${encodeURIComponent(portalUrl)}` : '/portal');
          }}
        >
          Open portal
        </button>
        <button
          type="button"
          class="btn btn-ghost btn-sm gap-1"
          onclick={recheckCaptivePortal}
          disabled={$loading['portal']}
        >
          {#if $loading['portal']}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <RefreshCw class="w-4 h-4" />
          {/if}
          Recheck
        </button>
      </div>
    </div>
  {/if}

  <div class="grid grid-cols-1 gap-4">
    <div class="card bg-base-100 shadow-sm border border-base-300">
      <div class="card-body">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0 lg:flex-1">
            <div class="lg:flex lg:items-center lg:justify-between lg:gap-8">
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
                    {[$systemStatus?.external_ip_city, $systemStatus?.external_ip_country].filter(Boolean).join(', ')}
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
              </div>
            {#if $systemStatus?.external_ip_isp || $systemStatus?.external_ip_asn || $systemStatus?.external_ip_timezone}
              <dl class="grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-1 gap-x-6 gap-y-2 mt-3 lg:mt-0 lg:w-72 lg:shrink-0 text-sm">
                {#if $systemStatus?.external_ip_isp}
                  <div class="min-w-0">
                    <dt class="text-base-content/60">ISP</dt>
                    <dd class="font-medium truncate" title={$systemStatus.external_ip_isp}>{$systemStatus.external_ip_isp}</dd>
                  </div>
                {/if}
                {#if $systemStatus?.external_ip_asn}
                  <div class="min-w-0">
                    <dt class="text-base-content/60">ASN</dt>
                    <dd class="font-medium truncate" title={$systemStatus.external_ip_asn}>{$systemStatus.external_ip_asn}</dd>
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

  {#if ethernetDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <EthernetPort class="w-5 h-5 text-primary" />
        Ethernet
      </h2>
      <div class="grid gap-4 md:grid-cols-2">
        {#each ethernetDevices as device}
          <div class="card bg-base-100 shadow-sm border border-base-300 card-hover">
            <div class="card-body">
              <div class="flex items-start justify-between mb-4">
                <div>
                  <h3 class="font-semibold">{device.interface}</h3>
                  <p class="text-sm text-base-content/60">{device.mac}</p>
                </div>
                <span class="badge {getStateBadge(device.state).class}">{getStateBadge(device.state).label}</span>
              </div>

              {#if device.state === 100}
                <div class="space-y-2 mb-4 p-3 rounded-lg bg-success/10">
                  <div class="flex items-center justify-between">
                    <div>
                      <p class="font-medium">Wired connection</p>
                      {#if device.ipv4?.addresses?.length}
                        <p class="text-xs font-mono text-base-content/70">{device.ipv4.addresses.map((a: { address: string; prefix: number }) => `${a.address}/${a.prefix}`).join(', ')}</p>
                      {/if}
                    </div>
                    <EthernetPort class="w-8 h-8 text-success" />
                  </div>
                  {#if device.ipv4?.gateway}
                    <p class="text-xs text-base-content/70"><span class="text-base-content/60">Gateway</span> <span class="font-mono">{device.ipv4.gateway}</span></p>
                  {/if}
                </div>
              {:else if device.state === 30}
                <div class="text-center py-4 text-base-content/50">
                  <EthernetPort class="w-12 h-12 mx-auto mb-2 opacity-50" />
                  <p>Cable connected, no active connection</p>
                </div>
              {:else}
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
                    onclick={() => handleCardConnect(device, '/connections', 'ethernet')}
                    disabled={connectingIface === device.interface}
                  >
                    Connect
                  </button>
                {:else}
                  <button type="button" class="btn btn-ghost flex-1" disabled>Please wait...</button>
                {/if}
                <button type="button" class="btn btn-ghost" onclick={() => navigate('/connections')}>Manage</button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}

  {#if wifiDevices.length > 0}
    <section>
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Wifi class="w-5 h-5 text-primary" />
        Wi-Fi Interfaces
      </h2>
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
                  <WifiOff class="w-12 h-12 mx-auto opacity-50" />
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
                    onclick={() => handleCardConnect(device, `/wifi?iface=${device.interface}`, 'wifi')}
                    disabled={connectingIface === device.interface}
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
                    onclick={() => handleCardConnect(device, '/connections', 'modem')}
                    disabled={connectingIface === device.interface}
                  >
                    Connect
                  </button>
                {:else}
                  <button type="button" class="btn btn-ghost flex-1" disabled>Please wait...</button>
                {/if}
                <button type="button" class="btn btn-ghost" onclick={() => navigate('/connections')}>Manage</button>
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
