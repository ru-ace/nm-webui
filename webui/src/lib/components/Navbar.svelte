<script lang="ts">
  import { onMount } from 'svelte';
  import { currentPath, navigate } from '$lib/stores/router';
  import { Wifi, Monitor, Server, Settings, Menu, Sun, Moon, MonitorCog } from 'lucide-svelte';
  import { activeTheme, cycleTheme, initTheme, themeMode } from '$lib/stores/theme';

  onMount(initTheme);

  const navItems = [
    { href: '/', label: 'Dashboard', icon: Monitor },
    { href: '/wifi', label: 'Wi-Fi', icon: Wifi },
    { href: '/devices', label: 'Devices', icon: Server },
    { href: '/connections', label: 'Profiles', icon: Settings }
  ];

  $: themeLabel = $themeMode === 'system' ? `System (${ $activeTheme })` : $themeMode === 'dark' ? 'Dark' : 'Light';
</script>

<nav class="navbar bg-base-100 shadow-sm border-b border-base-300 sticky top-0 z-50">
  <div class="navbar-start">
    <button class="btn btn-ghost btn-square lg:hidden" aria-label="Menu">
      <Menu class="w-6 h-6" />
    </button>
    <button class="navbar-brand" onclick={() => navigate('/')}>
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
        onclick={() => navigate(item.href)}
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
    <div class="dropdown dropdown-end">
      <button tabindex="0" class="btn btn-ghost btn-circle avatar" aria-label="User menu">
        <div class="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-content font-medium">
          NW
        </div>
      </button>
      <ul class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
        <li><button class="flex items-center gap-2 w-full" onclick={() => navigate('/connections')}><Settings class="w-4 h-4" /> Profiles</button></li>
        <li><button class="flex items-center gap-2 w-full" onclick={() => navigate('/')}>Logout</button></li>
      </ul>
    </div>
  </div>
</nav>
