<svelte:window on:click={() => {}} />

<script lang="ts">
  import { onMount } from 'svelte';
  import { initEventSource, loadSystemStatus, loadDevices, loadConnections, loadCaptivePortal } from '$lib/stores/app';
  import Layout from './routes/+layout.svelte';
  import Dashboard from './routes/+page.svelte';
  import WifiPage from './routes/wifi/+page.svelte';
  import DevicesPage from './routes/devices/+page.svelte';
  import ConnectionsPage from './routes/connections/+page.svelte';
  import PortalPage from './routes/portal/+page.svelte';
  import PasswordModal from './routes/wifi/PasswordModal.svelte';
  import { currentPath } from '$lib/stores/router';

  onMount(() => {
    loadSystemStatus();
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
</script>

<Layout>
  <div class="min-h-screen">
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
    {:else}
      <Dashboard />
    {/if}
  </div>
</Layout>

<PasswordModal />