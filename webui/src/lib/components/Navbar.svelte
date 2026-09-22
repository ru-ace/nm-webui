<script lang="ts">
  import { onMount } from 'svelte';
  import { currentPath, navigate } from '$lib/stores/router';
  import { Wifi, Monitor, Server, Settings, Sun, Moon, MonitorCog, Globe, Power, Loader2, ChevronDown } from '@lucide/svelte';
  import { activeTheme, cycleTheme, initTheme, themeMode } from '$lib/stores/theme';
  import { connectivityStatus, features, systemStatus } from '$lib/stores/app';
  import { headerAction } from '$lib/stores/header';

  onMount(initTheme);

  let menuOpen = false;

  $: navItems = [
    { href: '/', label: 'Dashboard', icon: Monitor },
    { href: '/wifi', label: 'Wi-Fi', icon: Wifi },
    { href: '/portal', label: 'Portal', icon: Globe },
    { href: '/devices', label: 'Devices', icon: Server },
    { href: '/connections', label: 'Profiles', icon: Settings },
    // The Power section only exists when a power-action-password is configured.
    ...($features?.power ? [{ href: '/power', label: 'Power', icon: Power }] : [])
  ];

  // Current page header shown in the navbar on mobile.
  $: currentNav =
    navItems.find((i) => $currentPath === i.href || (i.href !== '/' && $currentPath.startsWith(i.href))) ??
    navItems[0];

  // Compact connectivity dot for the Dashboard header on mobile.
  $: statusDotClass = {
    online: 'bg-success',
    limited: 'bg-warning',
    portal: 'bg-warning'
  }[$connectivityStatus] || 'bg-error';

  $: themeLabel = $themeMode === 'system' ? `System (${ $activeTheme })` : $themeMode === 'dark' ? 'Dark' : 'Light';

  function go(path: string) {
    menuOpen = false;
    navigate(path);
  }
</script>

<nav class="navbar relative bg-base-100 shadow-sm border-b border-base-300 sticky top-0 z-50">
  <div class="navbar-start">
    <button class="navbar-brand" onclick={() => go('/')}>
      <div class="flex items-center gap-2">
        <Wifi class="w-6 h-6 text-primary" />
        <span class="font-bold text-lg hidden lg:inline">{$systemStatus?.hostname || 'nm-webui'}</span>
      </div>
    </button>
  </div>
  <div class="navbar-center hidden lg:flex flex-wrap gap-1 px-2">
    {#each navItems as item}
      <button
        class:btn-active={$currentPath === item.href || (item.href !== '/' && $currentPath.startsWith(item.href))}
        class="btn btn-ghost gap-2 px-3 py-2 text-sm"
        onclick={() => go(item.href)}
      >
        <svelte:component this={item.icon} class="w-5 h-5" />
        <span>{item.label}</span>
      </button>
    {/each}
  </div>
  <button
    class="navbar-center lg:hidden flex items-center justify-center gap-2 px-2 min-w-0"
    type="button"
    aria-label="Open menu"
    aria-expanded={menuOpen}
    title="Menu"
    onclick={() => (menuOpen = !menuOpen)}
  >
    <svelte:component this={currentNav.icon} class="w-5 h-5 text-primary shrink-0" />
    <h1 class="text-lg font-bold truncate min-w-0">{currentNav.label}</h1>
    {#if $currentPath === '/'}
      <span
        class="h-2.5 w-2.5 rounded-full shrink-0 {statusDotClass}"
        title={`Connectivity: ${$connectivityStatus}`}
      ></span>
    {/if}
    <ChevronDown
      class="w-4 h-4 shrink-0 transition-transform {menuOpen ? 'rotate-180' : ''}"
      aria-hidden="true"
    />
  </button>
  <div class="navbar-end">
    {#if $headerAction}
      <button
        class="btn btn-primary btn-circle shadow-sm lg:hidden"
        type="button"
        onclick={$headerAction.onClick}
        disabled={$headerAction.disabled}
        aria-label={$headerAction.label}
        title={$headerAction.label}
      >
        {#if $headerAction.loading}
          <Loader2 class="w-5 h-5 animate-spin" />
        {:else}
          <svelte:component this={$headerAction.icon} class="w-5 h-5" />
        {/if}
      </button>
    {/if}
    <button
      class="btn btn-ghost btn-circle"
      type="button"
      onclick={cycleTheme}
      aria-label={`Switch theme: ${themeLabel}`}
      title={`${themeLabel}. Click to switch.`}
    >
      {#if $themeMode === 'system'}
        <MonitorCog class="w-5 h-5" />
      {:else if $activeTheme === 'dark'}
        <Moon class="w-5 h-5" />
      {:else}
        <Sun class="w-5 h-5" />
      {/if}
    </button>
  </div>
  {#if menuOpen}
    <div class="absolute left-0 right-0 top-full border-b border-base-300 bg-base-100 p-3 shadow-lg lg:hidden">
      <div class="menu gap-2 p-0 items-center w-full">
        {#each navItems as item}
          <button
            class:active={$currentPath === item.href || (item.href !== '/' && $currentPath.startsWith(item.href))}
            class="flex w-full max-w-sm mx-auto min-h-12 items-center justify-center gap-3 rounded-lg px-4 py-3 text-base font-medium"
            type="button"
            onclick={() => go(item.href)}
          >
            <svelte:component this={item.icon} class="h-6 w-6" />
            <span>{item.label}</span>
          </button>
        {/each}
      </div>
    </div>
  {/if}
</nav>
