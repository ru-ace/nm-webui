<script lang="ts">
  import { onMount } from 'svelte';
  import { initEventSource, loadSystemStatus, loadDevices, loadConnections } from '$lib/stores/app';
  import Navbar from '$lib/components/Navbar.svelte';
  import Toast from '$lib/components/Toast.svelte';

  onMount(() => {
    loadSystemStatus();
    loadDevices();
    loadConnections();
    const es = initEventSource();
    return () => es.close();
  });
</script>

<div class="min-h-screen flex flex-col">
  <Navbar />
  <main class="flex-1 p-4 md:p-6 overflow-auto">
    <slot />
  </main>
  <Toast />
</div>