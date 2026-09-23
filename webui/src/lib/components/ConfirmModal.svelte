<script lang="ts">
  import { fly } from 'svelte/transition';

  export let show = false;
  export let title = 'Confirm';
  export let message = '';
  export let confirmLabel = 'Confirm';
  export let cancelLabel = 'Cancel';
  export let onConfirm: (() => void) | undefined = undefined;
  export let onCancel: (() => void) | undefined = undefined;

  function cancel() {
    onCancel?.();
    show = false;
  }

  function confirm() {
    onConfirm?.();
    show = false;
  }
</script>

{#if show}
  <div class="modal modal-open" role="dialog">
    <div class="modal-box relative" in:fly={{ y: 20, duration: 200 }} out:fly={{ y: -20, duration: 150 }}>
      <h3 class="font-bold text-lg mb-2">{title}</h3>
      {#if message}
        <p class="text-sm text-base-content/60 mb-4">{message}</p>
      {/if}
      <div class="modal-action mt-4">
        <button type="button" class="btn btn-ghost" onclick={cancel}>{cancelLabel}</button>
        <button type="button" class="btn btn-error" onclick={confirm}>{confirmLabel}</button>
      </div>
    </div>
    <button type="button" class="modal-backdrop" onclick={cancel} aria-hidden="true"></button>
  </div>
{/if}

<style>
  /* Same fixes as ConnectionModal: the modal is a sibling inside `space-y-*`
     containers, whose margin-top shifts the fixed overlay. The right band
     comes from daisyUI's root scroll-lock reserving a scrollbar gutter. */
  :global(.modal) {
    margin: 0 !important;
    width: 100vw;
  }
  :global(:root:has(:is(.modal-open, .modal:target, .modal-toggle:checked + .modal, .modal[open]))) {
    scrollbar-gutter: auto !important;
  }
</style>