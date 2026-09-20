<script lang="ts">
  import { X } from 'lucide-svelte';
  import { toast } from '../stores/app';
  import { fly } from 'svelte/transition';
</script>

{#if $toast}
  <div
    class="fixed bottom-4 right-4 z-[100] animate-in slide-in-from-bottom-4 duration-300"
    in:fly={{ y: 100, duration: 300 }}
    out:fly={{ y: -100, duration: 200 }}
    role="alert"
  >
    <div class="alert shadow-lg flex items-center gap-3 min-w-[280px] max-w-md"
      class:alert-success={$toast.type === 'success'}
      class:alert-error={$toast.type === 'error'}
      class:alert-info={$toast.type === 'info'}
    >
      <span class="flex-1">{$toast.message}</span>
      <button class="btn btn-ghost btn-xs" onclick={() => toast.set(null)}>
        <X class="w-4 h-4" />
      </button>
    </div>
  </div>
{/if}