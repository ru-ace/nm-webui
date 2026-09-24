<script lang="ts">
  import { ChevronDown, ChevronUp, Settings, WifiOff, Link2, Unlink2, Loader2, Copy, Wifi, Cable, Database, Server, Smartphone } from '@lucide/svelte';
  import { disconnectDevice, connectDevice, connectWithPicker, connectingIface } from '$lib/stores/app';
  import { navigate } from '$lib/stores/router';

  export let device: any;
  export let loadingMap: any;

  let expanded = false;

  function getDeviceIconType(type: string, wireless: boolean, modem: boolean) {
    if (wireless) return 'wifi';
    if (modem) return 'modem';
    if (type === 'ethernet') return 'cable';
    if (type === 'bridge') return 'database';
    return 'server';
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

  function formatIPs(config: any) {
    if (!config?.addresses?.length) return '—';
    return config.addresses.map((a: any) => `${a.address}/${a.prefix}`).join(', ');
  }

  function formatDNS(config: any) {
    if (!config?.nameservers?.length) return '—';
    return config.nameservers.join(', ');
  }

  $: stateBadge = getStateBadge(device.state);
  $: deviceIconType = getDeviceIconType(device.type_name, device.wireless, !!device.modem);
  // Kind of the profile-picker flow (null keeps the plain "up" for exotic
  // device types that the picker modal has no icon/copy for).
  $: pickerKind = device.wireless ? 'wifi' : device.modem ? 'modem' : device.type_name === 'ethernet' ? 'ethernet' : null;
  // Manage route: the Wi-Fi section for wireless cards, Profiles for everything else.
  $: manageRoute = device.wireless ? `/wifi?iface=${device.interface}` : '/connections';
</script>

<div class="card bg-base-100 shadow-sm border border-base-300 card-hover">
  <div class="card-body">
    <div class="flex items-start justify-between mb-4">
      <div class="flex items-center gap-3">
        <div class="p-2 bg-primary/10 rounded-lg">
          <svelte:component this={deviceIconType === 'wifi' ? Wifi : deviceIconType === 'cable' ? Cable : deviceIconType === 'modem' ? Smartphone : deviceIconType === 'database' ? Database : Server} class="w-5 h-5" />
        </div>
        <div>
          <h3 class="font-semibold">{device.interface}</h3>
          <p class="text-sm text-base-content/60">{device.mac}</p>
        </div>
      </div>
      <span class="badge {stateBadge.class}">{stateBadge.label}</span>
    </div>

    <div class="space-y-3 text-sm">
      <div class="grid grid-cols-2 gap-2">
        <div><span class="text-base-content/60">Driver</span><br><span class="font-medium">{device.driver || '—'}</span></div>
        <div><span class="text-base-content/60">MTU</span><br><span class="font-medium">{device.mtu}</span></div>
        <div><span class="text-base-content/60">IPv4</span><br><span class="font-mono text-xs">{formatIPs(device.ipv4)}</span></div>
        <div><span class="text-base-content/60">Gateway</span><br><span class="font-mono text-xs">{device.ipv4?.gateway || '—'}</span></div>
        <div class="col-span-2"><span class="text-base-content/60">DNS</span><br><span class="font-mono text-xs">{formatDNS(device.ipv4)}</span></div>
        {#if device.ipv6?.addresses?.length}
          <div class="col-span-2"><span class="text-base-content/60">IPv6</span><br><span class="font-mono text-xs">{formatIPs(device.ipv6)}</span></div>
        {/if}
      </div>
    </div>

    {#if device.modem}
      <div class="mt-3 pt-3 border-t border-base-300 space-y-2 text-sm">
        <div class="flex items-center gap-2 flex-wrap">
          <span class="text-base-content/60">Mobile network</span>
          <span class="font-medium">{device.modem.operator_name || '—'}</span>
          {#if device.modem.access_tech_name}
            <span class="badge badge-primary badge-sm">{device.modem.access_tech_name}</span>
          {/if}
          {#if device.modem.signal}
            <span class="badge badge-ghost badge-sm">Signal {device.modem.signal.percent}%</span>
          {/if}
          {#if device.modem.state_name && device.state !== 100}
            <span class="badge badge-ghost badge-sm">Modem: {device.modem.state_name}</span>
          {/if}
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div><span class="text-base-content/60">APN</span><br><span class="font-mono text-xs">{device.modem.apn || '—'}</span></div>
          <div><span class="text-base-content/60">Modem</span><br><span class="font-medium">{device.modem.manufacturer || ''} {device.modem.model || '—'}</span></div>
          <div><span class="text-base-content/60">IMEI</span><br><span class="font-mono text-xs">{device.modem.imei || '—'}</span></div>
          <div><span class="text-base-content/60">Capabilities</span><br><span class="text-xs">{device.modem.capabilities_text || '—'}</span></div>
          {#if device.modem.sim}
            <div class="col-span-2"><span class="text-base-content/60">SIM</span><br><span class="font-mono text-xs">
              {#if device.modem.sim.operator_name}{device.modem.sim.operator_name}{/if}
              {#if device.modem.sim.iccid}({device.modem.sim.iccid}){/if}
            </span></div>
          {/if}
        </div>
      </div>
    {/if}

    <div class="flex gap-2 mt-4 pt-4 border-t border-base-300">
      {#if device.state === 100}
        <button
          type="button"
          class="btn btn-error btn-sm flex-1 gap-1"
          onclick={() => disconnectDevice(device.interface)}
          disabled={loadingMap[`disconnect-${device.interface}`]}
        >
          <Unlink2 class="w-4 h-4" /> Disconnect
        </button>
      {:else if device.state === 30}
        {#if pickerKind}
          <button
            type="button"
            class="btn btn-primary btn-sm flex-1 gap-1"
            onclick={() => connectWithPicker(device.interface, manageRoute, pickerKind)}
            disabled={$connectingIface === device.interface}
          >
            {#if $connectingIface === device.interface}
              <Loader2 class="w-4 h-4 animate-spin" />
            {:else}
              <Link2 class="w-4 h-4" />
            {/if}
            Connect
          </button>
        {:else}
          <button
            type="button"
            class="btn btn-primary btn-sm flex-1 gap-1"
            onclick={() => connectDevice(device.interface)}
            disabled={loadingMap[`connect-${device.interface}`]}
          >
            <Link2 class="w-4 h-4" /> Connect
          </button>
        {/if}
      {/if}
      <button type="button" class="btn btn-ghost btn-sm gap-1" onclick={() => navigate(manageRoute)}>
        <Settings class="w-4 h-4" /> Manage
      </button>
      <button class="btn btn-ghost btn-sm" onclick={() => expanded = !expanded}>
        {#if expanded}
          <ChevronUp class="w-4 h-4" />
        {:else}
          <ChevronDown class="w-4 h-4" />
        {/if}
      </button>
    </div>

    {#if expanded}
      <div class="mt-4 pt-4 border-t border-base-300 space-y-2 text-xs">
        <div class="flex justify-between"><span class="text-base-content/60">Path</span><span class="font-mono">{device.path}</span></div>
        <div class="flex justify-between"><span class="text-base-content/60">Type</span><span>{device.type_name}</span></div>
        <div class="flex justify-between"><span class="text-base-content/60">Managed</span><span>{device.managed ? 'Yes' : 'No'}</span></div>
        {#if device.active_connection}
          <div class="flex justify-between"><span class="text-base-content/60">Active Connection</span><span class="font-mono truncate max-w-[150px]">{device.active_connection}</span></div>
        {/if}
        <div class="flex justify-between"><span class="text-base-content/60">Auto-connect</span><span>{device.autoconnect ? 'Yes' : 'No'}</span></div>
      </div>
    {/if}
  </div>
</div>