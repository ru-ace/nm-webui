// Zero-dependency mock backend for documentation screenshots.
// Serves the built SPA (internal/webui/dist) plus the /api/v1 endpoints the
// frontend calls, with fictitious "travel router" data. No NetworkManager.
//
//   import { startMockServer } from './mock-server.mjs';
//   const server = await startMockServer({ distDir });   // ephemeral port
//   await server.close();
//
// Standalone:  node mock-server.mjs [PORT]
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { extname, join, normalize } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import {
  SYSTEM_STATUS,
  SYSTEM_STATUS_PORTAL,
  DEVICES,
  WIFI_STATUS,
  WIFI_NETWORKS,
  CONNECTIONS,
  DEVICE_CONNECTIONS,
  CAPTIVE_PORTAL,
  CAPTIVE_PORTAL_PORTAL,
  PORTAL_PAGE,
} from './mock-data.mjs';

export const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.webp': 'image/webp',
  '.ico': 'image/x-icon',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
  '.map': 'application/json',
};

const json = (res, body, status = 200) => {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
};

export function startMockServer({ distDir = join(REPO_ROOT, 'internal', 'webui', 'dist'), host = '127.0.0.1', port = 0 } = {}) {
  // Served data set: 'online' (default) or 'portal'. Screenshot captures switch
  // it per state via POST /__mock/mode so the dashboard-portal frame shows the
  // captive-portal banner while every other state stays online.
  let mode = 'online';
  const server = createServer(async (req, res) => {
    const url = new URL(req.url, `http://${host}:${port}`);
    const p = url.pathname;

    // Test-only control endpoint: switch which data set is served.
    if (p === '/__mock/mode' && req.method === 'POST') {
      let body = '';
      for await (const chunk of req) body += chunk;
      let parsed = {};
      try { parsed = JSON.parse(body || '{}'); } catch { parsed = {}; }
      const next = url.searchParams.get('mode') || parsed.mode;
      if (next !== 'online' && next !== 'portal') {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'mode must be online|portal' }));
        return;
      }
      mode = next;
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ ok: true, mode }));
      return;
    }

    // SSE stream
    if (p === '/api/v1/events') {
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      });
      res.write('retry: 5000\n\n');
      // Keep the push verdict consistent with the served data so the
      // connectivity badge and any banner agree in the captures.
      const connectivity = mode === 'portal'
        ? { status: 'portal', connectivity: 2 }
        : { status: 'online', connectivity: 4 };
      res.write(`event: connectivity_changed\ndata: ${JSON.stringify(connectivity)}\n\n`);
      const timer = setInterval(() => res.write(': keepalive\n\n'), 15000);
      req.on('close', () => clearInterval(timer));
      return;
    }

    // REST API
    if (p.startsWith('/api/v1/')) {
      const apiPath = p.slice('/api/v1'.length);
      const method = req.method;

      if (apiPath === '/captive-portal/proxy') {
        const target = url.searchParams.get('url') || '';
        if (req.method === 'POST') {
          let body = '';
          for await (const chunk of req) body += chunk;
          res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
          res.end('<html><body><h1>Signed in — session active for 24 h</h1><p><a href="http://captive.apple.com/hotspot-detect.html">Continue</a></p></body></html>');
          return;
        }
        if (/^https?:\/\//i.test(target)) {
          res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
          res.end(PORTAL_PAGE);
          return;
        }
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'invalid url' }));
        return;
      }

      if (p === '/api/v1/system/features') return json(res, { power: true });
      if (p === '/api/v1/system/status') return json(res, mode === 'portal' ? SYSTEM_STATUS_PORTAL : SYSTEM_STATUS);
      if (p === '/api/v1/system/external-ip/refresh' && method === 'POST') return json(res, { status: 'ok' });
      if (p === '/api/v1/system/power' && method === 'POST') return json(res, { status: 'ok' });
      if (p === '/api/v1/system/captive-portal') return json(res, mode === 'portal' ? CAPTIVE_PORTAL_PORTAL : CAPTIVE_PORTAL);
      if (p === '/api/v1/system/captive-portal/check' && method === 'POST') return json(res, mode === 'portal' ? CAPTIVE_PORTAL_PORTAL : CAPTIVE_PORTAL);

      if (p === '/api/v1/devices' && method === 'GET') return json(res, { devices: DEVICES });
      const devMatch = p.match(/^\/api\/v1\/devices\/([^/]+)(\/(disconnect|up))?$/);
      if (devMatch) {
        const iface = decodeURIComponent(devMatch[1]);
        const action = devMatch[3];
        const dev = DEVICES.find((d) => d.interface === iface);
        if (!dev) return json(res, { error: 'device not found' }, 404);
        if (action) return method === 'POST' ? json(res, { status: 'ok' }) : json(res, { error: 'method not allowed' }, 405);
        return json(res, dev);
      }

      // Profiles applicable to a device (NetworkManager AvailableConnections).
      const devConnsMatch = p.match(/^\/api\/v1\/devices\/([^/]+)\/connections$/);
      if (devConnsMatch) {
        const iface = decodeURIComponent(devConnsMatch[1]);
        if (method !== 'GET') return json(res, { error: 'method not allowed' }, 405);
        return json(res, { connections: DEVICE_CONNECTIONS[iface] || [] });
      }

      const wifiMatch = p.match(/^\/api\/v1\/wifi\/([^/]+)\/(networks|status|scan|connect)$/);
      if (wifiMatch) {
        const iface = decodeURIComponent(wifiMatch[1]);
        const what = wifiMatch[2];
        if (what === 'networks' && method === 'GET') return json(res, { networks: WIFI_NETWORKS });
        if (what === 'status' && method === 'GET') return json(res, WIFI_STATUS[iface] || { connected: false });
        if (what === 'scan' && method === 'POST') return json(res, { status: 'ok' });
        if (what === 'connect' && method === 'POST') return json(res, { status: 'ok', ssid: 'HomeWiFi' });
      }

      if (p === '/api/v1/connections') {
        if (method === 'GET') return json(res, { connections: CONNECTIONS });
        if (method === 'POST') return json(res, CONNECTIONS[0]);
      }
      const connMatch = p.match(/^\/api\/v1\/connections\/([^/]+)(\/(up|down))?$/);
      if (connMatch) {
        const action = connMatch[2];
        if (action) return method === 'POST' ? json(res, { status: 'ok', path: '/dev/null' }) : json(res, { error: 'method not allowed' }, 405);
        if (method === 'PUT') return json(res, CONNECTIONS[0]);
        if (method === 'DELETE') return json(res, { status: 'deleted' });
        if (method === 'GET') return json(res, CONNECTIONS[0]);
      }

      return json(res, { error: 'not found' }, 404);
    }

    // Static SPA with index.html fallback
    let filePath = p === '/' ? '/index.html' : p;
    const full = normalize(join(distDir, filePath));
    if (!full.startsWith(distDir)) {
      res.writeHead(403);
      res.end('forbidden');
      return;
    }
    try {
      const data = await readFile(full);
      res.writeHead(200, { 'Content-Type': MIME[extname(full)] || 'application/octet-stream' });
      res.end(data);
    } catch {
      try {
        const data = await readFile(join(distDir, 'index.html'));
        res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
        res.end(data);
      } catch {
        res.writeHead(500);
        res.end('frontend dist missing — run: cd webui && npm run build');
      }
    }
  });

  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, host, () => {
      const actual = server.address().port;
      server.removeListener('error', reject);
      console.log(`[screenshots] mock server on http://${host}:${actual} (dist: ${distDir})`);
      resolve({
        server,
        port: actual,
        close: () => new Promise((ok) => server.close(ok)),
      });
    });
  });
}

// Standalone mode: node mock-server.mjs [PORT]
const isMain = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;
if (isMain) {
  const port = Number(process.argv[2]) || 18080;
  await startMockServer({ port });
}