// One-command screenshot regeneration:
//   node run.mjs
//
// 1. verifies the frontend is built (internal/webui/dist/index.html),
// 2. starts the mock backend on an ephemeral port,
// 3. captures all screenshots into docs/screenshots,
// 4. shuts the mock down,
// 5. validates docs/SCREENSHOTS.md links.
//
// Overrides: NM_WEBUI_DIST=<dir>   frontend dist dir
//            OUT=<dir>             where to write PNGs (default docs/screenshots)
//            CHROMIUM_PATH=...     browser binary (see capture.mjs)
//            --states a,b,c        capture only some states (also forwarded)
import { access } from 'node:fs/promises';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { startMockServer, REPO_ROOT } from './mock-server.mjs';
import { captureAll } from './capture.mjs';
import { checkLinks } from './check-links.mjs';

const distDir = process.env.NM_WEBUI_DIST || join(REPO_ROOT, 'internal', 'webui', 'dist');
const outDir = process.env.OUT || join(REPO_ROOT, 'docs', 'screenshots');

async function main() {
  try {
    await access(join(distDir, 'index.html'));
  } catch {
    console.error(
      '[screenshots] frontend is not built: ' + join(distDir, 'index.html') +
      '\nRun first:  cd webui && npm ci && npm run build   (or: make screenshots)'
    );
    process.exit(1);
  }

  const mock = await startMockServer({ distDir });
  try {
    await captureAll({ baseUrl: `http://127.0.0.1:${mock.port}`, outDir });
  } finally {
    await mock.close();
  }
  await checkLinks();
}

const isMain = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;
if (isMain) {
  main().catch((e) => {
    console.error(e);
    process.exit(1);
  });
}