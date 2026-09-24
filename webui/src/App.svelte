<svelte:window on:click={() => {}} />

<script lang="ts">
  import { onMount } from 'svelte';
  import { initEventSource, loadSystemStatus, loadDevices, loadConnections, loadCaptivePortal, loadFeatures, systemStatus } from '$lib/stores/app';
  import Layout from './routes/+layout.svelte';
  import Dashboard from './routes/+page.svelte';
  import WifiPage from './routes/wifi/+page.svelte';
  import DevicesPage from './routes/devices/+page.svelte';
  import ConnectionsPage from './routes/connections/+page.svelte';
  import PortalPage from './routes/portal/+page.svelte';
  import PowerPage from './routes/power/+page.svelte';
  import PasswordModal from './routes/wifi/PasswordModal.svelte';
  import ProfileSelectModal from './lib/components/ProfileSelectModal.svelte';
  import WifiProfileModal from './lib/components/WifiProfileModal.svelte';
  import { currentPath } from '$lib/stores/router';

  onMount(() => {
    loadSystemStatus();
    loadFeatures();
    loadDevices();
    loadConnections();
    loadCaptivePortal();
    const es = initEventSource();

    function handleRoute() {
      currentPath.set(window.location.pathname);
    }

    handleRoute();
    window.addEventListener('popstate', handleRoute);

    return () => {
      window.removeEventListener('popstate', handleRoute);
      es.close();
    };
  });

  // The hostname is known only at runtime (system status), so replace the
  // static "nm-webui" tab title once the device reports it.
  $: document.title = $systemStatus?.hostname || 'nm-webui';
</script>

<Layout>
  <div class="flex-1 min-h-0 flex flex-col">
    {#if $currentPath === '/'}
      <Dashboard />
    {:else if $currentPath === '/wifi'}
      <WifiPage />
    {:else if $currentPath === '/devices'}
      <DevicesPage />
    {:else if $currentPath === '/connections'}
      <ConnectionsPage />
    {:else if $currentPath === '/portal'}
      <PortalPage />
    {:else if $currentPath === '/power'}
      <PowerPage />
    {:else}
      <Dashboard />
    {/if}
  </div>
</Layout>

<PasswordModal />
<ProfileSelectModal />
<WifiProfileModal />