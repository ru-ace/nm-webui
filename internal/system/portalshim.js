// Portal-browser request routing shim.
//
// This snippet is injected as the first child of <body> in proxied portal
// pages (allowJS mode) so it runs before any deferred module script of the
// SPA. The browser resolves everything against the canonical <base href="/">,
// which points at this listener's own origin -- a listener that only carries
// the proxy endpoints. Any relative request the app makes (config.json, API
// calls, OAuth discovery on a different host) would therefore land on the
// portal listener and 404. Instead of JS byte-patching the app, fetch and
// XMLHttpRequest are patched here to route HTTP(S) targets through the proxy
// endpoint, reconstructing what the upstream really answered; HTMLScriptElement.src
// is patched too so lazy module chunks (dynamic import()) follow the same path.
//
// Rules:
//   - non-HTTP schemes and self-origin proxy calls are left untouched
//   - relative URLs (the <base href="/"> artifact) are rebased onto the real
//     portal site from the nm-final-url marker and proxied
//   - absolute cross-origin URLs are proxied as-is
//
// The same code runs under an opaque sandbox (fallback mode): location.origin
// is "null" there, so the proxy base is derived from location.href instead.

(function () {
  if (window.__nmPortalShim) return;
  window.__nmPortalShim = true;

  var PROXY_PATH = "/api/v1/captive-portal/proxy";
  var LOC_ORIGIN = "null";
  try { LOC_ORIGIN = location.origin; } catch (e) {}

  function docOrigin() {
    if (LOC_ORIGIN !== "null") return LOC_ORIGIN;
    try { return new URL(location.href).origin; } catch (e) { return ""; }
  }

  // Relative URLs in the document resolve against the document base URL, not
  // against location.href (which is the proxy endpoint here). The rewriter
  // canonicalises <base href="/">, so relative references resolve to this
  // listener's root -- matching what the browser will actually request.
  function docBaseURL() {
    try {
      var b = document.querySelector('base[href]');
      var h = b && b.getAttribute("href");
      if (h) return new URL(h, location.href).href;
    } catch (e) {}
    return location.href;
  }

  function resolveURL(raw) {
    try { return new URL(raw, docBaseURL()); } catch (e) { return null; }
  }

  function pageBase() {
    try {
      var m = document.querySelector('meta[name="nm-final-url"]');
      var v = m && m.getAttribute("content");
      if (v && /^https?:\/\//i.test(v)) return v;
    } catch (e) {}
    return docOrigin() + "/";
  }

  function rebase(u) {
    var pb;
    try { pb = new URL(pageBase()); } catch (e) { return null; }
    return pb.origin + u.pathname + u.search;
  }

  function proxiedTarget(raw) {
    if (typeof raw !== "string" || raw === "" || raw.charAt(0) === "#") return null;
    var u = resolveURL(raw);
    if (!u) return null;
    if (u.protocol !== "http:" && u.protocol !== "https:") return null;
    if (u.pathname === PROXY_PATH) return null;               // never re-proxy
    if (!/^[a-z][a-z0-9+.\-]*:/i.test(raw)) return rebase(u); // <base href="/"> artifact
    // Any absolute URL that lands back on the serving origin (this listener,
    // or the admin origin under the opaque-sandbox fallback) is a base-href
    // artifact too -- e.g. lazy chunk URLs the browser's module loader builds
    // from <base href="/">. Rebase it onto the real portal site.
    if (u.origin === docOrigin()) return rebase(u);
    return u.href;                                            // absolute cross-origin
  }

  function proxyURL(target) {
    return docOrigin() + PROXY_PATH + "?url=" + encodeURIComponent(target);
  }

  // ---- fetch ----
  var nativeFetch = window.fetch;
  if (typeof nativeFetch === "function") {
    window.fetch = function (input, init) {
      var raw = null, req = null;
      if (typeof input === "string") raw = input;
      else if (input instanceof URL) raw = input.href;
      else if (input && typeof input.url === "string") { raw = input.url; req = input; }
      if (!raw) return nativeFetch.apply(this, arguments);
      var target = proxiedTarget(raw);
      if (!target) return nativeFetch.apply(this, arguments);
      var opts = {};
      opts.method = (init && init.method) || (req && req.method) || "GET";
      var h = (init && init.headers) || (req && req.headers);
      if (h) opts.headers = h;
      var body = (init && "body" in init) ? init.body
        : (req && "body" in req && typeof req.clone === "function") ? req.clone().body : undefined;
      if (body !== undefined && body !== null && opts.method !== "GET" && opts.method !== "HEAD") {
        opts.body = body;
        opts.duplex = "half"; // stream bodies need it in Chromium; ignored elsewhere
      }
      return nativeFetch(proxyURL(target), opts).then(function (resp) {
        return resp.arrayBuffer().then(function (buf) {
          var hd = new Headers();
          var ct = resp.headers.get("content-type");
          if (ct) hd.set("content-type", ct);
          return new Response(buf, { status: resp.status, statusText: resp.statusText, headers: hd });
        });
      });
    };
  }

  // ---- XMLHttpRequest ----
  var NativeXHR = window.XMLHttpRequest;
  if (typeof NativeXHR === "function") {
    function PortalXHR() {
      var x = new NativeXHR();
      var self = this;
      var et;
      if (typeof EventTarget === "function") {
        et = new EventTarget();
      } else {
        et = {
          _l: {},
          addEventListener: function (t, f) { (this._l[t] = this._l[t] || []).push(f); },
          removeEventListener: function (t, f) {
            var a = this._l[t];
            if (a) { var i = a.indexOf(f); if (i !== -1) a.splice(i, 1); }
          },
          dispatchEvent: function (e) {
            var a = (this._l[e.type] || []).slice();
            for (var i = 0; i < a.length; i++) { try { a[i].call(this, e); } catch (err) {} }
            return true;
          }
        };
      }
      function fire(type) {
        try { et.dispatchEvent(new Event(type)); } catch (e) {}
        var h = self["on" + type];
        if (h) { try { h.call(self); } catch (e) {} }
      }
      this.readyState = 0;
      this.status = 0;
      this.statusText = "";
      this.response = null;
      this.responseText = "";
      this.responseURL = "";
      x.onreadystatechange = function () {
        self.readyState = x.readyState;
        self.status = x.status;
        self.statusText = x.statusText;
        self.responseURL = x.responseURL || "";
        if (self.readyState === 4) {
          var rt = x.responseType;
          if (rt === "" || rt === "text") { try { self.responseText = x.responseText; } catch (e) {} }
          self.response = x.response;
        }
        fire("readystatechange");
      };
      x.onload = function () { fire("load"); fire("loadend"); };
      x.onerror = function () { fire("error"); fire("loadend"); };
      x.onabort = function () { fire("abort"); fire("loadend"); };
      x.ontimeout = function () { fire("timeout"); fire("loadend"); };
      this.addEventListener = function (t, f, c) { et.addEventListener(t, f, c); };
      this.removeEventListener = function (t, f, c) { et.removeEventListener(t, f, c); };
      this.open = function (method, url, async, user, pass) {
        var target = proxiedTarget(url);
        x.open(method, target ? proxyURL(target) : url, async === undefined ? true : !!async, user, pass);
      };
      this.send = function (body) { x.send(body); };
      this.abort = function () { x.abort(); };
      this.setRequestHeader = function (k, v) { x.setRequestHeader(k, v); };
      this.getResponseHeader = function (k) { return x.getResponseHeader(k); };
      this.getAllResponseHeaders = function () { return x.getAllResponseHeaders(); };
      this.overrideMimeType = function (t) { try { x.overrideMimeType(t); } catch (e) {} };
      Object.defineProperties(this, {
        responseType: { get: function () { return x.responseType; }, set: function (v) { x.responseType = v; } },
        timeout: { get: function () { return x.timeout; }, set: function (v) { x.timeout = v; } },
        withCredentials: { get: function () { return x.withCredentials; }, set: function (v) { x.withCredentials = v; } },
        upload: { get: function () { return x.upload; } }
      });
    }
    window.XMLHttpRequest = PortalXHR;
  }

  // ---- lazy module chunks ----
  // SPA bundles (webpack/rollup) lazy-load route chunks with dynamic import().
  // The browser's module loader resolves those URLs itself -- not through
  // fetch/XHR, so the patched network layer above never sees them: a chunk URL
  // resolves against the canonical <base href="/"> to this listener's root and
  // 404s. The loader assigns HTMLScriptElement.src before fetching, so patching
  // that setter reroutes chunk requests through the proxy like every other
  // subresource. Static <script src> tags of the rewritten page are parsed
  // directly and never hit this setter.
  var ScriptProto = (typeof HTMLScriptElement !== "undefined" && HTMLScriptElement.prototype) || null;
  var srcDesc = ScriptProto && Object.getOwnPropertyDescriptor(ScriptProto, "src") || null;
  if (srcDesc && typeof srcDesc.set === "function") {
    Object.defineProperty(ScriptProto, "src", {
      get: srcDesc.get,
      set: function (v) {
        var raw = typeof v === "string" ? v : String(v);
        var target = proxiedTarget(raw);
        srcDesc.set.call(this, target ? proxyURL(target) : v);
      },
      enumerable: !!srcDesc.enumerable,
      configurable: true
    });
  }
})();