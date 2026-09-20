<script lang="ts">
  import { X, Wifi, Cable, Database, Loader2, Smartphone } from 'lucide-svelte';
  import { fly } from 'svelte/transition';
  import { createConnection, updateConnection } from '$lib/stores/app';

  export let show = false;
  export let editing: any = null;

  // Groups the profile editor knows how to configure. Profiles of any other
  // NM type (vpn, loopback, ...) can still be renamed and get IP settings
  // changed, but their type is preserved untouched on save.
  const typeOptions = ['ethernet', 'wifi', 'bridge', 'gsm'] as const;

  let form = {
    id: '',
    interface: '',
    type: 'ethernet',
    ssid: '',
    password: '',
    apn: '',
    number: '*99#',
    username: '',
    pin: '',
    autoconnect: true,
    ipv4: { method: 'auto', address: '', prefix: 24, gateway: '', dns: '' },
    ipv6: { method: 'auto', address: '', prefix: 64, gateway: '', dns: '' }
  };
  let submitting = false;

  function normalizeType(typeRaw: string, typeName?: string): string {
    if (typeName && (typeOptions as readonly string[]).includes(typeName)) return typeName;
    if (typeRaw === '802-11-wireless') return 'wifi';
    if (typeRaw === '802-3-ethernet') return 'ethernet';
    if (typeRaw === 'bridge') return 'bridge';
    if (typeRaw === 'gsm' || typeRaw === 'cdma') return 'gsm';
    return typeRaw || 'ethernet';
  }

  $: if (editing) {
    form = {
      id: editing.id,
      interface: editing.interface || '',
      type: normalizeType(editing.type || '', editing.type_name),
      ssid: editing.ssid || '',
      password: '',
      apn: editing.apn || '',
      number: editing.number || '*99#',
      username: editing.username || '',
      pin: '',
      autoconnect: editing.autoconnect,
      ipv4: { method: editing.ipv4_method, address: editing.static4?.address || '', prefix: editing.static4?.prefix || 24, gateway: editing.static4?.gateway || '', dns: editing.static4?.dns?.join(', ') || '' },
      ipv6: { method: editing.ipv6_method, address: editing.static6?.address || '', prefix: editing.static6?.prefix || 64, gateway: editing.static6?.gateway || '', dns: editing.static6?.dns?.join(', ') || '' }
    };
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    // Never rewrite the type of profiles the editor does not support (vpn,
    // loopback, ...): the backend maps unknown types to ethernet otherwise.
    const known = !editing || (typeOptions as readonly string[]).includes(form.type as any);
    const data: any = {
      id: form.id,
      interface: form.interface,
      ...(known ? { type: form.type } : {}),
      ...(form.type === 'wifi' ? { ssid: form.ssid, password: form.password } : {}),
      ...(form.type === 'gsm'
        ? { apn: form.apn, number: form.number || '*99#', username: form.username, password: form.password, pin: form.pin }
        : {}),
      autoconnect: form.autoconnect,
      ipv4: form.ipv4.method === 'manual'
        ? { method: 'manual', address: form.ipv4.address, prefix: Number(form.ipv4.prefix), gateway: form.ipv4.gateway, dns: form.ipv4.dns.split(',').map(s => s.trim()).filter(Boolean) }
        : { method: form.ipv4.method },
      ipv6: form.ipv6.method === 'manual'
        ? { method: 'manual', address: form.ipv6.address, prefix: Number(form.ipv6.prefix), gateway: form.ipv6.gateway, dns: form.ipv6.dns.split(',').map(s => s.trim()).filter(Boolean) }
        : { method: form.ipv6.method }
    };
    if (editing?.uuid) {
      updateConnection(editing.uuid, data);
    } else {
      createConnection(data);
    }
    close();
  }

  function close() {
    show = false;
    editing = null;
    form = { id: '', interface: '', type: 'ethernet', ssid: '', password: '', apn: '', number: '*99#', username: '', pin: '', autoconnect: true, ipv4: { method: 'auto', address: '', prefix: 24, gateway: '', dns: '' }, ipv6: { method: 'auto', address: '', prefix: 64, gateway: '', dns: '' } };
    submitting = false;
  }

  function getTypeIconType(type: string) {
    if (type === 'wifi') return 'wifi';
    if (type === 'ethernet') return 'cable';
    if (type === 'gsm') return 'modem';
    return 'database';
  }

  function getTypeLabel(type: string) {
    return ({ ethernet: 'Ethernet', wifi: 'Wi-Fi', bridge: 'Bridge', gsm: 'Mobile Broadband' } as Record<string, string>)[type] || type;
  }

  function getTypeIcon(type: string) {
    if (type === 'wifi') return Wifi;
    if (type === 'ethernet') return Cable;
    if (type === 'gsm') return Smartphone;
    return Database;
  }

  function nmTypeOf(type: string) {
    return ({ ethernet: '802-3-ethernet', wifi: '802-11-wireless', bridge: 'bridge', gsm: 'gsm' } as Record<string, string>)[type] || type;
  }
</script>

{#if show}
  <div class="modal modal-open" role="dialog">
    <div class="modal-box max-w-2xl" in:fly={{ y: 20, duration: 200 }} out:fly={{ y: -20, duration: 150 }}>
      <form onsubmit={handleSubmit}>
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-bold text-lg">{editing ? 'Edit Profile' : 'New Profile'}</h3>
          <button type="button" class="btn btn-ghost btn-circle" onclick={close}><X class="w-5 h-5" /></button>
        </div>

        <div class="grid gap-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <div class="label"><span class="label-text">Profile Name</span></div>
              <input bind:value={form.id} class="input input-bordered w-full" required placeholder="My Connection" />
            </div>
            <div>
              <div class="label"><span class="label-text">Interface (optional)</span></div>
              <input bind:value={form.interface} class="input input-bordered w-full" placeholder="eth0, wlan0" />
            </div>
          </div>

          <div>
            <div class="label"><span class="label-text">Type</span></div>
            {#if editing}
              <div class="flex flex-wrap items-center gap-2 py-1.5">
                <span class="badge badge-lg badge-primary gap-2">
                  <svelte:component this={getTypeIcon(form.type)} class="w-4 h-4" />
                  {getTypeLabel(form.type)}
                </span>
                <span class="font-mono text-sm text-base-content/60">{nmTypeOf(form.type)}</span>
                <span class="text-xs text-base-content/50">{typeOptions.includes(form.type as any) ? 'not changeable while editing' : 'name and IP settings can still be changed'}</span>
              </div>
            {:else}
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <button type="button" class="btn btn-outline gap-2" class:btn-primary={form.type === 'ethernet'} onclick={() => form.type = 'ethernet'}>
                  <Cable class="w-4 h-4" /> Ethernet
                </button>
                <button type="button" class="btn btn-outline gap-2" class:btn-primary={form.type === 'wifi'} onclick={() => form.type = 'wifi'}>
                  <Wifi class="w-4 h-4" /> Wi-Fi
                </button>
                <button type="button" class="btn btn-outline gap-2" class:btn-primary={form.type === 'bridge'} onclick={() => form.type = 'bridge'}>
                  <Database class="w-4 h-4" /> Bridge
                </button>
                <button type="button" class="btn btn-outline gap-2" class:btn-primary={form.type === 'gsm'} onclick={() => form.type = 'gsm'}>
                  <Smartphone class="w-4 h-4" /> Mobile
                </button>
              </div>
            {/if}
          </div>

          {#if form.type === 'wifi'}
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <div class="label"><span class="label-text">SSID</span></div>
                <input bind:value={form.ssid} class="input input-bordered w-full" required placeholder="Network name" />
              </div>
              <div>
                <div class="label"><span class="label-text">Password (optional)</span></div>
                <input bind:value={form.password} type="password" class="input input-bordered w-full" placeholder={editing ? 'Leave empty to keep current password' : 'Wi-Fi password'} />
              </div>
            </div>
          {/if}

          {#if form.type === 'gsm'}
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <div class="label"><span class="label-text">APN</span></div>
                <input bind:value={form.apn} class="input input-bordered w-full" required placeholder="internet, internet.yota" />
              </div>
              <div>
                <div class="label"><span class="label-text">Dial Number</span></div>
                <input bind:value={form.number} class="input input-bordered w-full" placeholder="*99#" />
              </div>
              <div>
                <div class="label"><span class="label-text">Username (optional)</span></div>
                <input bind:value={form.username} class="input input-bordered w-full" placeholder="gdata" />
              </div>
              <div>
                <div class="label"><span class="label-text">Password (optional)</span></div>
                <input bind:value={form.password} type="password" class="input input-bordered w-full" placeholder={editing ? 'Leave empty to keep current' : 'APN password'} />
              </div>
              <div>
                <div class="label"><span class="label-text">SIM PIN (optional)</span></div>
                <input bind:value={form.pin} type="password" class="input input-bordered w-full" placeholder={editing ? 'Leave empty to keep current' : '1234'} />
              </div>
            </div>
          {/if}

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 items-start">
            <fieldset class="fieldset min-w-0">
              <legend class="fieldset-legend">IPv4 Configuration</legend>
              <div class="grid grid-cols-1 gap-3">
                <select bind:value={form.ipv4.method} class="select select-bordered min-w-0 w-full">
                  <option value="auto">DHCP (Auto)</option>
                  <option value="manual">Static</option>
                  <option value="disabled">Disabled</option>
                  <option value="link-local">Link-local only</option>
                  <option value="shared">Shared</option>
                </select>
                {#if form.ipv4.method === 'manual'}
                  <div class="grid grid-cols-1 gap-3 min-w-0">
                    <div>
                      <div class="label"><span class="label-text">Address / Prefix</span></div>
                      <div class="flex gap-2">
                        <input bind:value={form.ipv4.address} class="input input-bordered flex-1 min-w-0" placeholder="192.168.1.100" required />
                        <input bind:value={form.ipv4.prefix} type="number" min="0" max="32" class="input input-bordered w-24 shrink-0" required />
                      </div>
                    </div>
                    <div>
                      <div class="label"><span class="label-text">Gateway</span></div>
                      <input bind:value={form.ipv4.gateway} class="input input-bordered w-full" placeholder="192.168.1.1" />
                    </div>
                    <div>
                      <div class="label"><span class="label-text">DNS (comma separated)</span></div>
                      <input bind:value={form.ipv4.dns} class="input input-bordered w-full" placeholder="1.1.1.1, 8.8.8.8" />
                    </div>
                  </div>
                {/if}
              </div>
            </fieldset>

            <fieldset class="fieldset min-w-0">
              <legend class="fieldset-legend">IPv6 Configuration</legend>
              <div class="grid grid-cols-1 gap-3">
                <select bind:value={form.ipv6.method} class="select select-bordered min-w-0 w-full">
                  <option value="auto">DHCP (Auto)</option>
                  <option value="manual">Static</option>
                  <option value="disabled">Disabled</option>
                  <option value="link-local">Link-local only</option>
                  <option value="shared">Shared</option>
                  <option value="ignore">Ignore</option>
                </select>
                {#if form.ipv6.method === 'manual'}
                  <div class="grid grid-cols-1 gap-3 min-w-0">
                    <div>
                      <div class="label"><span class="label-text">Address / Prefix</span></div>
                      <div class="flex gap-2">
                        <input bind:value={form.ipv6.address} class="input input-bordered flex-1 min-w-0" placeholder="2001:db8::1" required />
                        <input bind:value={form.ipv6.prefix} type="number" min="0" max="128" class="input input-bordered w-24 shrink-0" required />
                      </div>
                    </div>
                    <div>
                      <div class="label"><span class="label-text">Gateway</span></div>
                      <input bind:value={form.ipv6.gateway} class="input input-bordered w-full" placeholder="2001:db8::1" />
                    </div>
                    <div>
                      <div class="label"><span class="label-text">DNS (comma separated)</span></div>
                      <input bind:value={form.ipv6.dns} class="input input-bordered w-full" placeholder="2606:4700:4700::1111" />
                    </div>
                  </div>
                {/if}
              </div>
            </fieldset>
          </div>

          <div class="flex items-center gap-2">
            <input type="checkbox" bind:checked={form.autoconnect} id="autoconnect" class="checkbox checkbox-primary" />
            <label for="autoconnect" class="label-text">Connect automatically</label>
          </div>
        </div>

        <div class="modal-action mt-6">
          <button type="button" class="btn btn-ghost" onclick={close}>Cancel</button>
          <button type="submit" class="btn btn-primary gap-2" disabled={submitting}>
            <div class="w-4 h-4" class:animate-spin={submitting}><Loader2 class="w-4 h-4" /></div>
            {editing ? 'Update' : 'Create'}
          </button>
        </div>
      </form>
    </div>
    <button type="button" class="modal-backdrop" onclick={close} aria-hidden="true"></button>
  </div>
{/if}

<style>
  /* The modal is rendered as a later sibling inside `space-y-*` containers,
     whose margin-top ends up shifting the fixed overlay (top band). The
     right band comes from daisyUI's root scroll-lock reserving a scrollbar
     gutter while the overlay only spans the layout viewport width. */
  :global(.modal) {
    margin: 0 !important;
    width: 100vw;
  }
  :global(:root:has(:is(.modal-open, .modal:target, .modal-toggle:checked + .modal, .modal[open]))) {
    scrollbar-gutter: auto !important;
  }
</style>