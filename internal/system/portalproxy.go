package system

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// PortalProxyMaxDoc limits the size of proxied documents (HTML pages and
// pass-through resources).
const PortalProxyMaxDoc = 8 << 20 // 8 MiB

// ProxyBase is the API endpoint the portal browser calls back to. Rewritten
// URLs in proxied documents point here.
const ProxyBase = "/api/v1/captive-portal/proxy"

// proxyMethod contains the rewritten form helper markers injected into pages.
const (
	proxyFormURLKey    = "url"
	proxyFormMethodKey = "_method"
)

// FetchPortal fetches target through the detector's HTTP client so requests
// originate from the host (the Wi-Fi client the portal binds to). HTML
// documents are decoded to UTF-8 (charset sniffing) and rewritten so that
// links, forms, styles and images resolve back through the proxy; other
// resources pass through untouched. Returns the body, the effective
// Content-Type, the final URL after redirects and the upstream status.
func (d *Detector) FetchPortal(ctx context.Context, method, target string, form url.Values) (body []byte, contentType string, finalURL string, status int, err error) {
	u, err := validateURL(target)
	if err != nil {
		return nil, "", "", 0, err
	}
	if form != nil {
		// Fold submitted fields into the request: append to the query for
		// GET, send urlencoded for POST. The target's own query wins except
		// where the form overrides the same key.
		if method == http.MethodGet {
			q := u.Query()
			for k, vs := range form {
				for _, v := range vs {
					q.Add(k, v)
				}
			}
			u.RawQuery = q.Encode()
		}
	}

	var req *http.Request
	if method == http.MethodPost && form != nil {
		req, err = http.NewRequestWithContext(ctx, method, u.String(), strings.NewReader(form.Encode()))
		if err != nil {
			return nil, "", "", 0, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, "", "", 0, err
		}
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; nm-webui) AppleWebKit/537.36 (KHTML, like Gecko) nm-webui-portal/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("fetch %s: %w", target, err)
	}
	defer resp.Body.Close()

	finalURL = resp.Request.URL.String()
	status = resp.StatusCode
	ct := resp.Header.Get("Content-Type")
	raw, err := io.ReadAll(io.LimitReader(resp.Body, PortalProxyMaxDoc+1))
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("read %s: %w", target, err)
	}
	if int64(len(raw)) > PortalProxyMaxDoc {
		return nil, "", "", 0, fmt.Errorf("document too large")
	}

	if isHTMLResponse(ct, raw) {
		decoded, err := decodeToUTF8(raw, ct)
		if err != nil {
			return nil, "", "", 0, err
		}
		rewritten, err := RewritePortalHTML(decoded, finalURL, d.allowJS)
		if err != nil {
			return nil, "", "", 0, err
		}
		return rewritten, "text/html; charset=utf-8", finalURL, status, nil
	}
	// Stylesheets pass through, but their url(...) references must be pointed
	// back at the proxy so relative assets resolve from the SPA origin.
	if strings.Contains(strings.ToLower(ct), "text/css") {
		if base, err := validateURL(finalURL); err == nil {
			css := rewriteCSS(string(raw), base)
			return []byte(css), ct, finalURL, status, nil
		}
	}
	return raw, ct, finalURL, status, nil
}

// isHTMLResponse reports whether the serving content looks like an HTML page.
func isHTMLResponse(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml") {
		return true
	}
	head := strings.ToLower(string(body[:min(len(body), 512)]))
	return strings.HasPrefix(head, "<!doctype") || strings.HasPrefix(head, "<html") ||
		strings.HasPrefix(head, "<head") || strings.HasPrefix(head, "<form")
}

// decodeToUTF8 decodes the body charset declared in the Content-Type into
// UTF-8 so proxied documents can be rendered/reparsed reliably.
func decodeToUTF8(raw []byte, contentType string) ([]byte, error) {
	reader, err := charset.NewReader(bytes.NewReader(raw), contentType)
	if err != nil {
		// Unparseable charset declaration: fall back to raw bytes.
		return raw, nil
	}
	out, err := io.ReadAll(io.LimitReader(reader, PortalProxyMaxDoc))
	if err != nil {
		return nil, fmt.Errorf("decode charset: %w", err)
	}
	return out, nil
}

var (
	cssURLRe    = regexp.MustCompile(`(?i)url\(\s*(?:"([^"]*)"|'([^']*)'|([^'")]*))\)`)
	metaRefresh = regexp.MustCompile(`(?i)^(\s*[0-9.]+\s*;\s*url=)(.*)$`)
)

// skipURLPrefixes are URL schemes/forms that must never be proxied.
var skipURLPrefixes = []string{"javascript:", "mailto:", "tel:", "data:", "about:", "ws:", "wss:"}

// RewritePortalHTML rewrites a portal document served by baseURL so every
// http(s) URL resolves back through ProxyBase. <base> is canonicalised to
// <base href="/"> (framework SPAs such as Angular refuse to bootstrap without
// an APP_BASE_HREF, and the root-relative base is safe because every rewritten
// URL is an absolute ProxyBase reference), while <iframe>, <object> and
// <embed> are always dropped (they cannot survive rewriting and would load
// against the admin's own origin). Forms get hidden url/_method fields, styles
// and meta-refresh markers are adjusted, and srcset candidates are rewritten
// individually.
//
// When allowJS is false, <script> elements and inline event-handler
// attributes (on*) plus srcdoc are stripped as well — nothing executable
// survives, which is the safe mode for unknown renderers. When allowJS is
// true, scripts and handlers are kept for the sandboxed-iframe renderer and a
// small telemetry snippet is injected so the SPA can track navigation.
func RewritePortalHTML(doc []byte, baseURL string, allowJS bool) ([]byte, error) {
	base, err := validateURL(baseURL)
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(bytes.NewReader(doc))
	if err != nil {
		return nil, fmt.Errorf("parse portal html: %w", err)
	}

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			switch tag {
			case "base":
				// Keep a canonical <base href="/"> instead of dropping the tag:
				// Angular and other framework SPAs require <base>/APP_BASE_HREF
				// to bootstrap ("No base href set" crash), and the canonical
				// root-relative base is safe here because the rewriter turns
				// every http(s) reference into an absolute ProxyBase URL that
				// already resolves against the admin origin root. A target
				// hint set by the portal is preserved.
				target := ""
				for _, a := range n.Attr {
					if strings.EqualFold(a.Key, "target") && a.Val != "" {
						target = a.Val
					}
				}
				n.Attr = []html.Attribute{{Key: "href", Val: "/"}}
				if target != "" {
					n.Attr = append(n.Attr, html.Attribute{Key: "target", Val: target})
				}
				n.FirstChild = nil
				n.LastChild = nil
				return
			case "iframe", "object", "embed":
				// Frames and plugins would load in the admin's own origin.
				if n.Parent != nil {
					n.Parent.RemoveChild(n)
				}
				return // children were removed with the node
			case "script":
				if !allowJS {
					// In no-JS mode scripts cannot survive rewriting and would
					// run against the admin's own origin.
					if n.Parent != nil {
						n.Parent.RemoveChild(n)
					}
					return
				}
			case "style":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						c.Data = rewriteCSS(c.Data, base)
					}
				}
			case "meta":
				rewriteMetaRefresh(n, base)
			case "form":
				rewriteForm(n, base)
			}
			// Strip attributes that would execute code, override the rewritten
			// form target, or spawn a top-level window against the admin
			// origin: per-control form overrides always go; inline event
			// handlers and srcdoc only in no-JS mode. target/formtarget are
			// stripped in both modes so proxied content can never open its own
			// top-level tab on the admin origin.
			kept := n.Attr[:0]
			for _, a := range n.Attr {
				key := strings.ToLower(a.Key)
				if key == "formaction" || key == "formmethod" || key == "target" || key == "formtarget" {
					continue
				}
				if !allowJS && (strings.HasPrefix(key, "on") || key == "srcdoc") {
					continue
				}
				kept = append(kept, a)
			}
			n.Attr = kept
			rewriteAttr(n, "href", base, allowJS)
			rewriteAttr(n, "src", base, allowJS)
			rewriteAttr(n, "action", base, allowJS)
			rewriteAttr(n, "data-src", base, allowJS)
			rewriteAttr(n, "poster", base, allowJS)
			rewriteAttr(n, "background", base, allowJS)
			rewriteAttr(n, "srcset", base, allowJS)
		}
		// Capture the next sibling before descending: RemoveChild clears the
		// removed node's NextSibling, which would truncate this loop.
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			walk(c)
			c = next
		}
	}
	walk(root)

	if allowJS {
		// The sandboxed iframe runs under an opaque origin that the parent
		// cannot inspect, so keep the SPA address bar and history in sync via
		// postMessage. The snippet is safe: it only reports its own URL/title.
		// The final-url marker lets the SPA show the actual destination after
		// upstream redirects (the iframe never sees them; the Go client does).
		if head := findElement(root, "head"); head != nil {
			head.AppendChild(&html.Node{
				Type: html.ElementNode, Data: "meta",
				Attr: []html.Attribute{
					{Key: "name", Val: "nm-final-url"},
					{Key: "content", Val: base.String()},
				},
			})
		}
		if body := findElement(root, "body"); body != nil {
			body.AppendChild(&html.Node{
				Type: html.ElementNode, Data: "script",
				FirstChild: &html.Node{
					Type: html.TextNode, Data: telemetryScript,
				},
			})
		}
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return nil, fmt.Errorf("render portal html: %w", err)
	}
	return buf.Bytes(), nil
}

// findElement returns the first descendant element with the given tag name.
func findElement(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && strings.ToLower(c.Data) == tag {
			return c
		}
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// telemetryScript reports the portal document's location, the final upstream
// URL (from the injected nm-final-url marker) and its title back to the SPA
// after loads and history-triggering navigation, so the mini-browser chrome
// (address bar, back/forward) stays accurate while scripts run inside the
// sandboxed iframe.
const telemetryScript = `(function(){
  function report(){
    try {
      var fin = null;
      var m = document.querySelector('meta[name="nm-final-url"]');
      if (m) fin = m.getAttribute("content");
      parent.postMessage({__nmPortal: true, href: location.href, finalUrl: fin || null, title: document.title}, "*");
    } catch(e){}
  }
  function hook(proto, name){
    var orig = proto && proto[name];
    if (!orig) return;
    proto[name] = function(){
      var r = orig.apply(this, arguments);
      setTimeout(report, 0);
      return r;
    };
  }
  hook(history, "pushState");
  hook(history, "replaceState");
  window.addEventListener("popstate", report);
  window.addEventListener("load", report);
  if (document.readyState !== "loading") report();
})();`

// proxiedURL converts an absolute URL into a proxy endpoint reference.
func proxiedURL(target string) string {
	return ProxyBase + "?" + proxyFormURLKey + "=" + url.QueryEscape(target)
}

// resolveAndProxy resolves raw against base and returns the proxy reference,
// or "" when the value should stay untouched (fragments, non-http schemes,
// already-proxied references).
func resolveAndProxy(raw string, base *url.URL) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	for _, p := range skipURLPrefixes {
		if strings.HasPrefix(strings.ToLower(trimmed), p) {
			return ""
		}
	}
	if strings.HasPrefix(trimmed, ProxyBase+"?") {
		return ""
	}
	ref, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(ref)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return proxiedURL(resolved.String())
}

// rewriteAttr rewrites one attribute of n if present.
func rewriteAttr(n *html.Node, name string, base *url.URL, allowJS bool) {
	for i, a := range n.Attr {
		if a.Key != name {
			continue
		}
		if name == "srcset" {
			n.Attr[i].Val = rewriteSrcset(a.Val, base)
			return
		}
		trimmed := strings.TrimSpace(a.Val)
		if proxied := resolveAndProxy(a.Val, base); proxied != "" {
			n.Attr[i].Val = proxied
		} else if !allowJS && strings.HasPrefix(strings.ToLower(trimmed), "javascript:") {
			// In no-JS mode scripts do not survive rewriting; neutralise
			// javascript: hrefs so they cannot fire.
			n.Attr[i].Val = "#"
		}
		return
	}
}

// rewriteSrcset rewrites every "url descriptor" candidate in a srcset value.
func rewriteSrcset(v string, base *url.URL) string {
	parts := strings.Split(v, ",")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		if proxied := resolveAndProxy(fields[0], base); proxied != "" {
			fields[0] = proxied
		}
		parts[i] = strings.Join(fields, " ")
	}
	return strings.Join(parts, ", ")
}

// rewriteForm pins the form to the proxy endpoint and adds hidden fields so
// the SPA can reconstruct the original target and method.
func rewriteForm(n *html.Node, base *url.URL) {
	action := ""
	origMethod := "post"
	for _, a := range n.Attr {
		switch a.Key {
		case "action":
			action = a.Val
		case "method":
			if strings.EqualFold(a.Val, "get") {
				origMethod = "get"
			}
		}
	}

	// Resolve the original action (which may be relative) into an absolute
	// target; it rides the proxy query and the hidden url input.
	target := base.String()
	if action != "" {
		if p := resolveAndProxy(action, base); p != "" {
			if u, err := url.Parse(p); err == nil {
				if v := u.Query().Get(proxyFormURLKey); v != "" {
					if decoded, err := url.QueryUnescape(v); err == nil {
						target = decoded
					}
				}
			}
		}
	}
	if p := proxiedURL(target); p != "" {
		setAttr(n, "action", p)
	}
	setAttr(n, "method", "post")

	// Drop any pre-existing hidden fields we manage, then inject ours.
	filtered := n.Attr[:0]
	for _, a := range n.Attr {
		if a.Key == "name" && (a.Val == proxyFormURLKey || a.Val == proxyFormMethodKey) {
			continue
		}
		filtered = append(filtered, a)
	}
	n.Attr = filtered
	n.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "input",
		Attr: []html.Attribute{
			{Key: "type", Val: "hidden"}, {Key: "name", Val: proxyFormURLKey}, {Key: "value", Val: target},
		},
	})
	n.AppendChild(&html.Node{
		Type: html.ElementNode, Data: "input",
		Attr: []html.Attribute{
			{Key: "type", Val: "hidden"}, {Key: "name", Val: proxyFormMethodKey}, {Key: "value", Val: origMethod},
		},
	})
}

// setAttr sets or replaces an attribute value on a node.
func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

// rewriteMetaRefresh rewrites the url= part of a meta refresh element.
func rewriteMetaRefresh(n *html.Node, base *url.URL) {
	for _, a := range n.Attr {
		if !strings.EqualFold(a.Key, "http-equiv") || !strings.EqualFold(strings.TrimSpace(a.Val), "refresh") {
			continue
		}
		for j, ca := range n.Attr {
			if !strings.EqualFold(ca.Key, "content") {
				continue
			}
			m := metaRefresh.FindStringSubmatch(ca.Val)
			if len(m) != 3 {
				return
			}
			if proxied := resolveAndProxy(m[2], base); proxied != "" {
				n.Attr[j].Val = m[1] + proxied
			}
			return
		}
	}
}

// rewriteCSS rewrites url(...) references inside a <style> block.
func rewriteCSS(css string, base *url.URL) string {
	return cssURLRe.ReplaceAllStringFunc(css, func(m string) string {
		sub := cssURLRe.FindStringSubmatch(m)
		var content string
		for _, g := range sub[1:] {
			if g != "" {
				content = g
				break
			}
		}
		content = strings.TrimSpace(content)
		if content == "" || strings.HasPrefix(strings.ToLower(content), "data:") {
			return m
		}
		if proxied := resolveAndProxy(content, base); proxied != "" {
			return "url('" + proxied + "')"
		}
		return m
	})
}
