<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { captivePortal, loading, recheckCaptivePortal, portalProxyBase, loadFeatures } from '$lib/stores/app';
  import { NAVIGATE_EVENT, navigate } from '$lib/stores/router';
  import { headerAction } from '$lib/stores/header';
  import {
    AlertTriangle,
    CheckCircle,
    ExternalLink,
    Globe,
    Loader2,
    RefreshCw,
    X
  } from '@lucide/svelte';
  import { fly } from 'svelte/transition';

  const PROXY = '/api/v1/captive-portal/proxy';
  // Opaque sandbox (default): portal scripts run in an opaque origin, so they
  // can never touch the admin SPA, its cookies or its API.
  const SANDBOX_OPAQUE = 'allow-forms allow-scripts allow-popups allow-modals';
  // Distinct-origin sandbox: with the portal served by the dedicated proxy
  // listener (a separate origin, see portal_proxy_base) the frame becomes
  // same-origin only with itself, never with the admin, so allow-same-origin
  // is safe and framework SPAs get working localStorage on their own origin.
  const SANDBOX_DISTINCT = 'allow-forms allow-scripts allow-popups allow-modals allow-same-origin';

  let frameEl: HTMLIFrameElement | null = null;
  let frameSrc = '';
  let loadedProxyBase = '';
  let sandbox = SANDBOX_OPAQUE;
  let urlInput = '';
  let addressValue = '';
  let currentUrl = '';
  let portalTitle = '';
  let busy = false;
  let errorMessage = '';
  // "Open in new tab" bypasses the iframe sandbox (opaque origin), so it must
  // be explicitly re-confirmed every time. Two modes: proxy (default, portal
  // hostname resolved by the router) and direct (browser resolves over the
  // router's WAN — required for OAuth/SPA portals that cannot run through
  // the HTML proxy).
  let confirmOpenTab = false;
  let openTabConfirmed = false;
  let openTabMode: 'proxy' | 'direct' = 'proxy';
  let onlineNotice = false;

  function isHttp(target: string): boolean {
    return /^https?:\/\//i.test(target.trim());
  }

  // The proxy may live on the admin's own origin (dedicated listener
  // disabled) or on a separate origin served by PortalProxyHandler.
  function portalBase(): string {
    const base = get(portalProxyBase);
    if (base) return base.replace(/\/+$/, '');
    return window.location.origin;
  }

  function proxyUrl(target: string): string {
    return `${portalBase()}${PROXY}?url=${encodeURIComponent(target.trim())}`;
  }

  function targetOfQuery(u: URL): string {
    const t = u.searchParams.get('url') || '';
    return isHttp(t) ? t.trim() : '';
  }

  // Decode the upstream target out of a proxy href reported by telemetry.
  function targetOfProxyHref(href: string): string {
    try {
      const u = new URL(href, window.location.origin);
      if (u.pathname !== PROXY) return '';
      const t = u.searchParams.get('url') || '';
      return isHttp(t) ? t : '';
    } catch {
      return '';
    }
  }

  // Keep the browser bar pointing at /portal?url=<current> so a refresh or a
  // programmatic re-navigation re-opens the same page. replaceState only —
  // the SPA nav stack must not grow here.
  function reflectLocation(target: string) {
    const q = target ? `?url=${encodeURIComponent(target)}` : '';
    window.history.replaceState({}, '', `/portal${q}`);
  }

  // Point the iframe at the proxy. The nonce guarantees a fresh navigation
  // even when the target did not change (same proxy URL would not reload),
  // and the proxy ignores the extra query parameter.
  function loadInFrame(target: string) {
    currentUrl = target;
    addressValue = target;
    urlInput = target;
    portalTitle = '';
    errorMessage = '';
    onlineNotice = false;
    busy = true;
    reflectLocation(target);
    loadedProxyBase = portalBase();
    frameSrc = `${proxyUrl(target)}&r=${Date.now()}`;
  }

  // The sandbox and proxy base depend on the portal-proxy origin, which the
  // backend announces in /system/features after a (usually quick) boot fetch.
  $: sandbox = (() => {
    const base = $portalProxyBase;
    if (!base) return SANDBOX_OPAQUE;
    try {
      return new URL(base).origin !== window.location.origin ? SANDBOX_DISTINCT : SANDBOX_OPAQUE;
    } catch {
      return SANDBOX_OPAQUE;
    }
  })();

  // If a frame was already loaded before the features arrived (auto-open or a
  // direct navigation), re-point it at the dedicated proxy origin so portal
  // content enjoys the separate-origin sandbox rather than the fallback.
  $: if ($portalProxyBase !== '' && currentUrl && loadedProxyBase !== $portalProxyBase) {
    loadInFrame(currentUrl);
  }

  function onFrameLoad() {
    busy = false;
  }

  // Telemetry from the sandboxed portal frame (injected by the proxy): keep
  // the address bar and page title in sync with navigation that happens
  // inside the iframe (link clicks, form submits, redirects, meta refresh,
  // history API).
  function onPortalMessage(e: MessageEvent) {
    const d = e.data;
    if (!d || d.__nmPortal !== true) return;
    if (e.source !== frameEl?.contentWindow) return;
    const hrefTarget = targetOfProxyHref(String(d.href || ''));
    // The proxy embeds <meta name="nm-final-url"> when it followed upstream
    // redirects, so the mini-browser can show the actual destination (the
    // iframe URL itself never changes on a server-side redirect).
    const finalTarget =
      typeof d.finalUrl === 'string' && isHttp(d.finalUrl) ? d.finalUrl.trim() : '';
    const target = finalTarget || hrefTarget;
    if (!target) return;
    if (String(d.title)) portalTitle = String(d.title).trim();
    urlInput = target;
    addressValue = target;
    currentUrl = target;
    reflectLocation(target);
  }

  function go(target: string) {
    const t = (target || '').trim();
    if (!t) {
      errorMessage = 'Enter a URL, e.g. http://192.168.1.1/';
      return;
    }
    if (!isHttp(t)) {
      errorMessage = 'Only http:// and https:// URLs are supported.';
      return;
    }
    loadInFrame(t);
  }

  function submitGo() {
    go(urlInput);
  }

  async function recheck() {
    onlineNotice = false;
    await recheckCaptivePortal();
    const st = get(captivePortal);
    if (st?.state === 'portal' && st.portal_url && st.portal_url !== currentUrl) {
      go(st.portal_url);
    } else if (st?.state === 'online') {
      onlineNotice = true;
    }
  }

  function requestOpenTab() {
    if (!currentUrl) return;
    openTabMode = 'proxy';
    openTabConfirmed = false;
    confirmOpenTab = true;
  }

  function cancelOpenTab() {
    confirmOpenTab = false;
    openTabConfirmed = false;
  }

  function doOpenTab() {
    if (!openTabConfirmed || !currentUrl) return;
    // noopener+noreferrer keep the opened page detached from this window.
    const target = openTabMode === 'direct' ? currentUrl : proxyUrl(currentUrl);
    window.open(target, '_blank', 'noopener,noreferrer');
    confirmOpenTab = false;
    openTabConfirmed = false;
  }

  function initialTarget(): string {
    const fromQuery = targetOfQuery(new URL(window.location.href));
    if (fromQuery) return fromQuery;
    return get(captivePortal)?.portal_url || '';
  }

  function syncFromNavigation(ev?: Event) {
    const loc = ev ? (ev as CustomEvent<URL>).detail : new URL(window.location.href);
    const t = targetOfQuery(loc);
    if (t && t !== currentUrl) {
      go(t);
    }
  }

  // Open the detected portal automatically while this page is idle.
  $: if ($captivePortal?.state === 'portal' && !currentUrl && !busy && $captivePortal.portal_url) {
    go($captivePortal.portal_url);
  }

  // Mobile header action: icon-only Recheck button (hidden on desktop, where
  // the toolbar button is removed entirely).
  $: headerAction.set({
    label: 'Recheck portal',
    icon: RefreshCw,
    onClick: () => { void recheck(); },
    disabled: $loading['portal'],
    loading: $loading['portal']
  });

  onDestroy(() => headerAction.set(null));

  onMount(async () => {
    // The proxy base (dedicated portal-origin listener) must be known before
    // the first iframe load so the sandbox and URLs are chosen correctly.
    // It comes from /system/features; on failure we fall back to the admin's
    // own origin with the fully opaque sandbox.
    try {
      await loadFeatures();
    } catch {
      // fall back
    }
    const initial = initialTarget();
    if (initial) go(initial);

    function onNav(e: Event) {
      syncFromNavigation(e);
    }
    window.addEventListener(NAVIGATE_EVENT, onNav);
    window.addEventListener('popstate', onNav);
    window.addEventListener('message', onPortalMessage);
    return () => {
      window.removeEventListener(NAVIGATE_EVENT, onNav);
      window.removeEventListener('popstate', onNav);
      window.removeEventListener('message', onPortalMessage);
    };
  });
</script>

<div class="flex flex-col gap-2 flex-1 min-h-0">
  {#if onlineNotice}
    <div class="alert alert-success shadow-sm">
      <CheckCircle class="w-6 h-6 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold">Internet is available</p>
        <p class="text-sm text-success-content/70">The portal sign-in completed or the network no longer requires it.</p>
      </div>
      <button class="btn btn-sm" onclick={() => navigate('/')}>Go to dashboard</button>
    </div>
  {/if}

  <div class="card bg-base-100 shadow-sm border border-base-300 flex-1 min-h-0 flex flex-col">
    <div class="card-body p-3 gap-2 flex-1 min-h-0 flex flex-col">
      <!-- address bar -->
      <div class="flex gap-2">
        <div class="flex-1 relative">
          <Globe class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40 pointer-events-none" />
          <input
            bind:value={urlInput}
            onkeydown={(e) => { if (e.key === 'Enter') submitGo(); }}
            placeholder="http://192.168.1.1/ or any http(s) URL"
            class="input input-bordered input-sm w-full pl-9 font-mono text-sm"
            spellcheck="false"
          />
        </div>
        <button class="btn btn-primary btn-sm" onclick={submitGo} disabled={busy}>
          {#if busy}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            Go
          {/if}
        </button>
      </div>

      <!-- toolbar -->
      <div class="flex items-center gap-1 flex-wrap">
        <button
          type="button"
          class="btn btn-ghost btn-sm gap-1"
          onclick={requestOpenTab}
          disabled={!currentUrl}
          title="Open through the proxy in a new tab"
        >
          <ExternalLink class="w-4 h-4" />
          <span class="hidden lg:inline">Open in new tab</span>
        </button>
        <span class="flex-1 truncate text-xs font-mono text-base-content/50" title={addressValue}>
          {portalTitle || addressValue}
        </span>
      </div>

      {#if errorMessage}
        <div class="alert alert-error shadow-sm py-2">
          <span class="text-sm break-all">{errorMessage}</span>
        </div>
      {/if}

      <!-- content -->
      {#if !currentUrl && !busy}
        <div class="flex-1 flex items-center justify-center bg-base-200/50 rounded-lg border border-base-300 min-h-[40vh]">
          <div class="text-center px-4">
            <div class="flex flex-col items-center gap-3">
              <Globe class="w-10 h-10 text-base-content/40" />
              <p class="text-sm text-base-content/70 max-w-md">
                Enter a captive-portal URL above, or open the one detected by the probe.
              </p>
              {#if $captivePortal?.portal_url}
                <button class="btn btn-primary btn-sm" onclick={() => go($captivePortal.portal_url)}>
                  Open detected portal
                </button>
              {/if}
            </div>
          </div>
        </div>
      {:else}
        <!--
          The iframe always points at the proxy endpoint (never at the portal
          host directly), so portal hostnames are resolved only by the Go
          proxy on the host and never by the client's browser/DNS.
        -->
        <iframe
          bind:this={frameEl}
          src={frameSrc}
          sandbox={sandbox}
          referrerpolicy="no-referrer"
          onload={onFrameLoad}
          class="w-full flex-1 min-h-0 rounded-lg border border-base-300 bg-white"
          title="Portal"
        ></iframe>
      {/if}
    </div>
  </div>
</div>

{#if confirmOpenTab}
  <div class="modal modal-open" role="dialog" aria-modal="true">
    <div class="modal-box relative" in:fly={{ y: 20, duration: 200 }} out:fly={{ y: -20, duration: 150 }}>
      <button
        type="button"
        class="btn btn-ghost btn-circle absolute right-2 top-2"
        onclick={cancelOpenTab}
        aria-label="Close"
        title="Close"
      >
        <X class="w-5 h-5" />
      </button>
      <div class="flex items-start gap-3">
        <AlertTriangle class="w-6 h-6 shrink-0 text-warning" />
        <div>
          <h3 class="font-bold text-lg">Open in a new tab?</h3>
          <p class="text-sm text-base-content/70 mt-1">
            The page will open <span class="font-semibold">outside the sandbox</span>, so its scripts
            run with the same access to this admin interface as this window. Only open pages you trust.
          </p>
        </div>
      </div>
      <div class="mt-4 grid gap-2" role="radiogroup" aria-label="How to open">
        <label class="flex items-start gap-3 cursor-pointer border border-base-300 rounded-lg p-3 hover:bg-base-200/50 {openTabMode === 'proxy' ? 'border-primary' : ''}">
          <input type="radio" name="portal-open-mode" class="radio radio-sm mt-0.5" checked={openTabMode === 'proxy'} onchange={() => { openTabMode = 'proxy'; openTabConfirmed = false; }} />
          <span class="min-w-0">
            <span class="block text-sm font-semibold">Through the router proxy</span>
            <span class="block text-xs text-base-content/60 mt-0.5">
              The router resolves the portal hostname and relays the page. Default — works for
              login-form portals that do not require their own backend access from the browser.
            </span>
          </span>
        </label>
        <label class="flex items-start gap-3 cursor-pointer border border-base-300 rounded-lg p-3 hover:bg-base-200/50 {openTabMode === 'direct' ? 'border-primary' : ''}">
          <input type="radio" name="portal-open-mode" class="radio radio-sm mt-0.5" checked={openTabMode === 'direct'} onchange={() => { openTabMode = 'direct'; openTabConfirmed = false; }} />
          <span class="min-w-0">
            <span class="block text-sm font-semibold">Open the portal URL directly</span>
            <span class="block text-xs text-base-content/60 mt-0.5">
              Your browser connects to the portal host over the router's network. Required for
              OAuth/SPA portals (e.g. operator modems) that cannot run through the HTML proxy.
            </span>
          </span>
        </label>
      </div>
      <label class="flex items-start gap-2 mt-4 cursor-pointer">
        <input type="checkbox" bind:checked={openTabConfirmed} class="checkbox checkbox-sm mt-0.5" />
        <span class="text-sm">I understand — only open pages I trust this way.</span>
      </label>
      <div class="modal-action">
        <button type="button" class="btn btn-ghost" onclick={cancelOpenTab}>Cancel</button>
        <button
          type="button"
          class="btn btn-warning gap-2"
          onclick={doOpenTab}
          disabled={!openTabConfirmed}
        >
          <ExternalLink class="w-4 h-4" />
          Open
        </button>
      </div>
    </div>
    <button type="button" class="modal-backdrop" onclick={cancelOpenTab} aria-hidden="true"></button>
  </div>
{/if}