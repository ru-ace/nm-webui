<script lang="ts">
  import { onMount } from 'svelte';
  import { currentPath, navigate } from '$lib/stores/router';
  import { Wifi, Monitor, Server, Settings, Menu, Sun, Moon, MonitorCog, Globe, Power } from '@lucide/svelte';
  import { activeTheme, cycleTheme, initTheme, themeMode } from '$lib/stores/theme';
  import { features } from '$lib/stores/app';

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

  $: themeLabel = $themeMode === 'system' ? `System (${ $activeTheme })` : $themeMode === 'dark' ? 'Dark' : 'Light';

  function go(path: string) {
    menuOpen = false;
    navigate(path);
  }
</script>

<nav class="navbar relative bg-base-100 shadow-sm border-b border-base-300 sticky top-0 z-50">
  <div class="navbar-start">
    <button
      class="btn btn-ghost btn-square lg:hidden"
      type="button"
      aria-label="Menu"
      aria-expanded={menuOpen}
      onclick={() => menuOpen = !menuOpen}
    >
      <Menu class="w-6 h-6" />
    </button>
    <button class="navbar-brand" onclick={() => go('/')}>
      <div class="flex items-center gap-2">
        <Wifi class="w-6 h-6 text-primary" />
        <span class="font-bold text-lg">nm-webui</span>
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
  <div class="navbar-end">
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
      <div class="menu gap-2 p-0">
        {#each navItems as item}
          <button
            class:active={$currentPath === item.href || (item.href !== '/' && $currentPath.startsWith(item.href))}
            class="flex min-h-12 items-center gap-4 rounded-lg px-4 py-3 text-base font-medium"
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
