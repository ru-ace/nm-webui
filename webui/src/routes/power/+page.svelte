<script lang="ts">
  import { api } from '$lib/api/client';
  import { features } from '$lib/stores/app';
  import { AlertTriangle, Eye, EyeOff, Loader2, Lock, Power, RotateCcw, X } from '@lucide/svelte';
  import { fly } from 'svelte/transition';

  type PowerAction = 'reboot' | 'poweroff';

  let password = '';
  let showPassword = false;
  let pending: PowerAction | null = null;
  let confirmed = false;
  let submitting = false;
  let errorMsg = '';
  let actionSent = false;
  let sentAction: PowerAction | null = null;

  $: enabled = $features?.power === true;
  $: notLoaded = $features === null;
  $: actionLabel = pending === 'poweroff' ? 'Power off' : 'Reboot';
  $: canSubmit = !!password.trim() && !actionSent;

  function open(action: PowerAction) {
    if (actionSent) return;
    pending = action;
    confirmed = false;
    errorMsg = '';
  }

  function cancel() {
    if (submitting) return;
    pending = null;
    confirmed = false;
    errorMsg = '';
  }

  async function confirmAction() {
    if (!pending || !confirmed || submitting || actionSent) return;
    submitting = true;
    errorMsg = '';
    try {
      if (pending === 'reboot') {
        await api.power.reboot(password);
      } else {
        await api.power.poweroff(password);
      }
      sentAction = pending;
      actionSent = true; // the host is going away; lock the UI
    } catch (e: any) {
      errorMsg = e?.message || 'Power action failed';
      submitting = false;
    }
  }
</script>

<div class="space-y-6 max-w-3xl mx-auto">
  <div class="flex items-center gap-2">
    <Power class="w-7 h-7 text-warning" />
    <h1 class="text-2xl font-bold">Power</h1>
  </div>

  {#if notLoaded}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body items-center gap-3 text-base-content/60">
        <Loader2 class="w-6 h-6 animate-spin" />
        <span>Loading…</span>
      </div>
    </div>
  {:else if !enabled}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body items-center gap-3 text-center text-base-content/60">
        <Power class="w-10 h-10 text-base-content/30" />
        <p>Power management is disabled.</p>
        <p class="text-sm max-w-md">
          Enable it by setting <code class="badge badge-ghost">power-action-password</code> in the config
          or by starting the service with <code class="badge badge-ghost">--power-action-password=…</code>.
        </p>
      </div>
    </div>
  {:else}
    <div class="card bg-base-100 border border-base-300 shadow-sm">
      <div class="card-body gap-5">
        <div>
          <h2 class="text-lg font-semibold">Restart or shut down the server</h2>
          <p class="text-sm text-base-content/60 mt-1">
            Enter the power password, then confirm in the dialog. The server will become unavailable.
          </p>
        </div>

        <div class="relative">
          <input
            bind:value={password}
            type={showPassword ? 'text' : 'password'}
            class="input input-bordered w-full pr-12"
            placeholder="Power password"
            disabled={actionSent}
          />
          <div class="absolute right-1 top-1/2 -translate-y-1/2">
            <button
              type="button"
              class="btn btn-ghost btn-square"
              onclick={() => showPassword = !showPassword}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
              title={showPassword ? 'Hide password' : 'Show password'}
            >
              {#if showPassword}<EyeOff class="w-5 h-5" />{:else}<Eye class="w-5 h-5" />{/if}
            </button>
          </div>
        </div>

        <div class="flex items-center justify-between gap-3">
          <button
            type="button"
            class="btn btn-error gap-2 shrink-0"
            disabled={!canSubmit}
            onclick={() => open('poweroff')}
          >
            <Power class="w-4 h-4" />
            Power off
          </button>
          {#if actionSent}
            <span class="badge badge-warning gap-1">
              <Loader2 class="w-3 h-3 animate-spin" />
              action in progress
            </span>
          {/if}
          <button
            type="button"
            class="btn btn-warning gap-2 shrink-0"
            disabled={!canSubmit}
            onclick={() => open('reboot')}
          >
            <RotateCcw class="w-4 h-4" />
            Reboot
          </button>
        </div>

        <p class="text-sm text-base-content/60 flex items-center gap-1.5">
          <Lock class="w-4 h-4" />
          Reboot and Power off both require the power password and an explicit confirmation.
        </p>
      </div>
    </div>
  {/if}
</div>

{#if pending}
  <div class="modal modal-open" role="dialog" aria-modal="true">
    <div class="modal-box relative" in:fly={{ y: 20, duration: 200 }} out:fly={{ y: -20, duration: 150 }}>
      <button
        type="button"
        class="btn btn-ghost btn-circle absolute right-2 top-2"
        onclick={cancel}
        disabled={submitting}
        aria-label="Close"
        title="Close"
      >
        <X class="w-5 h-5" />
      </button>
      <div class="flex items-start gap-3">
        <AlertTriangle class="w-6 h-6 shrink-0 {pending === 'poweroff' ? 'text-error' : 'text-warning'}" />
        <div>
          <h3 class="font-bold text-lg">{actionLabel}</h3>
          <p class="text-sm text-base-content/70 mt-1">
            The server will become <span class="font-semibold">unavailable</span> and this connection
            will be lost.
            {#if pending === 'poweroff'}
              The device will shut down completely and must be powered back on manually.
            {:else}
              The device will restart and come back automatically.
            {/if}
          </p>
        </div>
      </div>

      {#if errorMsg}
        <div class="alert alert-error mt-4 py-2 text-sm">
          <AlertTriangle class="w-4 h-4 shrink-0" />
          <span>{errorMsg}</span>
        </div>
      {/if}

      <label class="flex items-start gap-2 mt-4 cursor-pointer">
        <input type="checkbox" bind:checked={confirmed} disabled={submitting} class="checkbox checkbox-sm mt-0.5" />
        <span class="text-sm">I understand the server will become unavailable.</span>
      </label>

      <div class="modal-action">
        <button type="button" class="btn btn-ghost" onclick={cancel} disabled={submitting}>Cancel</button>
        <button
          type="button"
          class="btn gap-2 {pending === 'poweroff' ? 'btn-error' : 'btn-warning'}"
          onclick={confirmAction}
          disabled={!confirmed || submitting}
        >
          {#if submitting}<Loader2 class="w-4 h-4 animate-spin" />{/if}
          {actionLabel}
        </button>
      </div>
    </div>
    <button type="button" class="modal-backdrop" onclick={cancel} aria-hidden="true"></button>
  </div>
{/if}

{#if actionSent}
  <div class="fixed inset-0 z-[100] flex flex-col items-center justify-center gap-4 bg-base-100/95">
    <Loader2 class="w-12 h-12 animate-spin {sentAction === 'poweroff' ? 'text-error' : 'text-warning'}" />
    <p class="text-lg font-semibold">
      {sentAction === 'poweroff' ? 'Shutting down the server…' : 'Rebooting the server…'}
    </p>
    <p class="text-sm text-base-content/60">The connection will be lost for a while.</p>
  </div>
{/if}