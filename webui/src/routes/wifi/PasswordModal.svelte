<script lang="ts">
  import { onMount } from 'svelte';
  import { X, Eye, EyeOff, Lock } from 'lucide-svelte';
  import { passwordModal } from '$lib/stores/modals';
  import { fly } from 'svelte/transition';

  let password = '';
  let showPassword = false;
  let inputRef: HTMLInputElement;

  function handleSubmit() {
    $passwordModal?.resolve(password);
    close();
  }

  function close() {
    $passwordModal?.resolve(null);
    password = '';
    showPassword = false;
    passwordModal.set(null);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleSubmit();
    if (e.key === 'Escape') close();
  }

  $: if ($passwordModal && inputRef) {
    setTimeout(() => inputRef?.focus(), 50);
  }

  $: eyeIcon = showPassword ? EyeOff : Eye;
</script>

{#if $passwordModal}
  <div class="modal modal-open" role="dialog">
    <div class="modal-box" in:fly={{ y: 20, duration: 200 }} out:fly={{ y: -20, duration: 150 }}>
      <h3 class="font-bold text-lg mb-4 flex items-center gap-2">
        <Lock class="w-5 h-5" />
        Connect to "{$passwordModal.ssid}"
      </h3>
      <p class="text-sm text-base-content/60 mb-4">Enter the Wi-Fi password</p>
      <div class="relative">
        <input
          bind:this={inputRef}
          bind:value={password}
          type={showPassword ? 'text' : 'password'}
          class="input input-bordered w-full pr-12"
          placeholder="Password"
          onkeydown={handleKeydown}
        />
        <button
          type="button"
          class="btn btn-ghost btn-square absolute right-1 top-1/2 -translate-y-1/2"
          onclick={() => showPassword = !showPassword}
        >
          <svelte:component this={eyeIcon} class="w-5 h-5" />
        </button>
      </div>
      <div class="modal-action mt-4">
        <button type="button" class="btn btn-ghost" onclick={close}>Cancel</button>
        <button class="btn btn-primary" onclick={handleSubmit} disabled={!password.trim()}>Connect</button>
      </div>
    </div>
    <button type="button" class="modal-backdrop" onclick={close} aria-hidden="true"></button>
  </div>
{/if}