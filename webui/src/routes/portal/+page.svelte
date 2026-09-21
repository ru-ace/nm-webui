<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { captivePortal, loading, recheckCaptivePortal } from '$lib/stores/app';
  import { NAVIGATE_EVENT, navigate } from '$lib/stores/router';
  import {
    ArrowLeft,
    ArrowRight,
    CheckCircle,
    ExternalLink,
    Globe,
    Home,
    Loader2,
    RefreshCw
  } from 'lucide-svelte';

  const PROXY = '/api/v1/captive-portal/proxy';
  // Sandboxed without allow-same-origin: portal scripts run in an opaque
  // origin, so they can never touch the admin SPA, its cookies or its API.
  const SANDBOX = 'allow-forms allow-scripts allow-popups allow-modals';

  let frameEl: HTMLIFrameElement | null = null;
  let frameSrc = '';
  let urlInput = '';
  let addressValue = '';
  let currentUrl = '';
  let portalTitle = '';
  let busy = false;
  let errorMessage = '';
  let onlineNotice = false;
  let historyStack: string[] = [];
  let historyIndex = -1;

  function isHttp(target: string): boolean {
    return /^https?:\/\//i.test(target.trim());
  }

  function proxyUrl(target: string): string {
    return `${PROXY}?url=${encodeURIComponent(target.trim())}`;
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

  function canGoBack() {
    return historyIndex > 0;
  }

  function canGoForward() {
    return historyIndex < historyStack.length - 1;
  }

  function pushHistory(target: string) {
    historyStack = historyStack.slice(0, historyIndex + 1);
    historyStack.push(target);
    historyIndex = historyStack.length - 1;
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
    frameSrc = `${proxyUrl(target)}&r=${Date.now()}`;
  }

  function onFrameLoad() {
    busy = false;
  }

  // Telemetry from the sandboxed portal frame (injected by the proxy): keep
  // the address bar, page title and the local history stack in sync with
  // navigation that happens inside the iframe (link clicks, form submits,
  // redirects, meta refresh, history API).
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
    if (target !== historyStack[historyIndex]) {
      pushHistory(target);
    }
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
    pushHistory(t);
    loadInFrame(t);
  }

  function submitGo() {
    go(urlInput);
  }

  function back() {
    if (!canGoBack()) return;
    historyIndex--;
    loadInFrame(historyStack[historyIndex]);
  }

  function forward() {
    if (!canGoForward()) return;
    historyIndex++;
    loadInFrame(historyStack[historyIndex]);
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

  onMount(() => {
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

<div class="flex flex-col gap-3">
  <div class="flex items-center gap-2">
    <button class="btn btn-ghost btn-sm gap-1" onclick={() => navigate('/')} aria-label="Dashboard">
      <ArrowLeft class="w-4 h-4" />
      Dashboard
    </button>
    <h1 class="text-lg font-semibold flex-1 truncate">Captive portal</h1>
  </div>

  {#if onlineNotice}
    <div class="alert alert-success shadow-sm">
      <CheckCircle class="w-6 h-6 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold">Internet is available</p>
        <p class="text-sm text-base-content/70">The portal sign-in completed or the network no longer requires it.</p>
      </div>
      <button class="btn btn-sm" onclick={() => navigate('/')}>Go to dashboard</button>
    </div>
  {/if}

  <div class="card bg-base-100 shadow-sm border border-base-300">
    <div class="card-body p-3 gap-2">
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
        <div class="flex gap-1">
          <button class="btn btn-ghost btn-sm" onclick={back} disabled={!canGoBack()} title="Back">
            <ArrowLeft class="w-4 h-4" />
          </button>
          <button class="btn btn-ghost btn-sm" onclick={forward} disabled={!canGoForward()} title="Forward">
            <ArrowRight class="w-4 h-4" />
          </button>
        </div>
        <button class="btn btn-ghost btn-sm gap-1" onclick={recheck} disabled={$loading['portal']}>
          {#if $loading['portal']}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <RefreshCw class="w-4 h-4" />
          {/if}
          Recheck
        </button>
        <a
          class="btn btn-ghost btn-sm gap-1 {currentUrl ? '' : 'btn-disabled'}"
          href={currentUrl ? proxyUrl(currentUrl) : '#'}
          target="_blank"
          rel="noopener noreferrer"
          title="Open through the proxy in a new tab"
        >
          <ExternalLink class="w-4 h-4" />
          Open in new tab
        </a>
        <button class="btn btn-ghost btn-sm gap-1" onclick={() => navigate('/')} title="Home">
          <Home class="w-4 h-4" />
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
        <div class="hero bg-base-200/50 rounded-lg border border-base-300 py-12">
          <div class="hero-content text-center">
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
          sandbox={SANDBOX}
          referrerpolicy="no-referrer"
          onload={onFrameLoad}
          class="w-full min-h-[55vh] max-h-[70vh] rounded-lg border border-base-300 bg-white"
          title="Captive portal"
        ></iframe>
      {/if}
    </div>
  </div>
</div>