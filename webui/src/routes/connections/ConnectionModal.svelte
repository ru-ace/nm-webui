<script lang="ts">
  import { X, Wifi, Cable, Database, Loader2, Smartphone } from 'lucide-svelte';
  import { fly } from 'svelte/transition';
  import { createConnection, updateConnection } from '$lib/stores/app';

  export let show = false;
  export let editing: any = null;

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

  $: if (editing) {
    form = {
      id: editing.id,
      interface: editing.interface || '',
      type: editing.type === '802-11-wireless' ? 'wifi' : (editing.type === '802-3-ethernet' ? 'ethernet' : (editing.type === 'gsm' ? 'gsm' : editing.type)),
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
    const data: any = {
      id: form.id,
      interface: form.interface,
      type: form.type,
      ...(form.type === 'wifi' && !editing ? { ssid: form.ssid, password: form.password } : {}),
      ...(form.type === 'gsm'
        ? { apn: form.apn, number: form.number || '*99#', username: form.username, password: form.password, pin: form.pin }
        : {}),
      autoconnect: form.autoconnect,
      ipv4: form.ipv4.method !== 'auto' ? { method: form.ipv4.method, address: form.ipv4.address, prefix: Number(form.ipv4.prefix), gateway: form.ipv4.gateway, dns: form.ipv4.dns.split(',').map(s => s.trim()).filter(Boolean) } : { method: 'auto' },
      ipv6: form.ipv6.method !== 'auto' ? { method: form.ipv6.method, address: form.ipv6.address, prefix: Number(form.ipv6.prefix), gateway: form.ipv6.gateway, dns: form.ipv6.dns.split(',').map(s => s.trim()).filter(Boolean) } : { method: 'auto' }
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
          <div class="grid grid-cols-2 gap-4">
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
            <div class="flex gap-2">
              <button type="button" class="btn btn-outline flex-1 gap-2" class:btn-primary={form.type === 'ethernet'} onclick={() => form.type = 'ethernet'} disabled={!!editing}>
                <Cable class="w-4 h-4" /> Ethernet
              </button>
              <button type="button" class="btn btn-outline flex-1 gap-2" class:btn-primary={form.type === 'wifi'} onclick={() => form.type = 'wifi'} disabled={!!editing}>
                <Wifi class="w-4 h-4" /> Wi-Fi
              </button>
              <button type="button" class="btn btn-outline flex-1 gap-2" class:btn-primary={form.type === 'bridge'} onclick={() => form.type = 'bridge'} disabled={!!editing}>
                <Database class="w-4 h-4" /> Bridge
              </button>
              <button type="button" class="btn btn-outline flex-1 gap-2" class:btn-primary={form.type === 'gsm'} onclick={() => form.type = 'gsm'} disabled={!!editing}>
                <Smartphone class="w-4 h-4" /> Mobile
              </button>
            </div>
          </div>

          {#if form.type === 'wifi' && !editing}
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="label"><span class="label-text">SSID</span></div>
                <input bind:value={form.ssid} class="input input-bordered w-full" required placeholder="Network name" />
              </div>
              <div>
                <div class="label"><span class="label-text">Password (optional)</span></div>
                <input bind:value={form.password} type="password" class="input input-bordered w-full" placeholder="Wi-Fi password" />
              </div>
            </div>
          {/if}

          {#if form.type === 'gsm' && !editing}
            <div class="grid grid-cols-2 gap-4">
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
                <input bind:value={form.password} type="password" class="input input-bordered w-full" placeholder="APN password" />
              </div>
              <div>
                <div class="label"><span class="label-text">SIM PIN (optional)</span></div>
                <input bind:value={form.pin} type="password" class="input input-bordered w-full" placeholder="1234" />
              </div>
            </div>
          {/if}

          <div class="grid grid-cols-2 gap-4">
            <fieldset class="fieldset">
              <legend class="fieldset-legend">IPv4 Configuration</legend>
              <div class="grid gap-3">
                <select bind:value={form.ipv4.method} class="select select-bordered">
                  <option value="auto">DHCP (Auto)</option>
                  <option value="manual">Static</option>
                  <option value="disabled">Disabled</option>
                </select>
                {#if form.ipv4.method === 'manual'}
                  <div class="grid grid-cols-2 gap-3">
                    <div>
                      <div class="label"><span class="label-text">Address / Prefix</span></div>
                      <div class="flex gap-2">
                        <input bind:value={form.ipv4.address} class="input input-bordered flex-1" placeholder="192.168.1.100" required />
                        <input bind:value={form.ipv4.prefix} type="number" min="0" max="32" class="input input-bordered w-20" required />
                      </div>
                    </div>
                    <div>
                      <div class="label"><span class="label-text">Gateway</span></div>
                      <input bind:value={form.ipv4.gateway} class="input input-bordered" placeholder="192.168.1.1" />
                    </div>
                    <div class="col-span-2">
                      <div class="label"><span class="label-text">DNS (comma separated)</span></div>
                      <input bind:value={form.ipv4.dns} class="input input-bordered" placeholder="1.1.1.1, 8.8.8.8" />
                    </div>
                  </div>
                {/if}
              </div>
            </fieldset>

            <fieldset class="fieldset">
              <legend class="fieldset-legend">IPv6 Configuration</legend>
              <div class="grid gap-3">
                <select bind:value={form.ipv6.method} class="select select-bordered">
                  <option value="auto">DHCP (Auto)</option>
                  <option value="manual">Static</option>
                  <option value="disabled">Disabled</option>
                </select>
                {#if form.ipv6.method === 'manual'}
                  <div class="grid grid-cols-2 gap-3">
                    <div>
                      <div class="label"><span class="label-text">Address / Prefix</span></div>
                      <div class="flex gap-2">
                        <input bind:value={form.ipv6.address} class="input input-bordered flex-1" placeholder="2001:db8::1" required />
                        <input bind:value={form.ipv6.prefix} type="number" min="0" max="128" class="input input-bordered w-20" required />
                      </div>
                    </div>
                    <div>
                      <div class="label"><span class="label-text">Gateway</span></div>
                      <input bind:value={form.ipv6.gateway} class="input input-bordered" placeholder="2001:db8::1" />
                    </div>
                    <div class="col-span-2">
                      <div class="label"><span class="label-text">DNS (comma separated)</span></div>
                      <input bind:value={form.ipv6.dns} class="input input-bordered" placeholder="2606:4700:4700::1111" />
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
