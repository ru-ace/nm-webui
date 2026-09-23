// Playwright capture of all documentation screenshots.
//
//   node capture.mjs --base http://127.0.0.1:PORT --out <dir> [--states a,b,c]
//
// Renders every state in light+dark themes on desktop (1280x800) and mobile
// (390x844) viewports. File names are stable so docs/SCREENSHOTS.md links
// never need to change:
//   <state>.png | <state>-dark.png | <state>-mobile.png | <state>-dark-mobile.png
//
// If a UI wait/selector fails (e.g. the frontend changed), that state throws
// with a clear message instead of silently producing a wrong screenshot.
//
// Chromium resolution order: $CHROMIUM_PATH → common system binaries →
// Playwright's bundled Chromium (`npx playwright install chromium`).
import { chromium } from 'playwright';
import { mkdirSync } from 'node:fs';
import { fileURLToPath, pathToFileURL } from 'node:url';

export const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));

const DESKTOP = { width: 1280, height: 800 };
const MOBILE = { width: 390, height: 844 };

function parseArgs() {
  const a = process.argv.slice(2);
  const get = (name) => {
    const i = a.indexOf(name);
    return i >= 0 ? a[i + 1] : undefined;
  };
  return {
    base: get('--base') || 'http://127.0.0.1:18080',
    out: get('--out') || `${REPO_ROOT}/docs/screenshots`,
    states: get('--states') ? get('--states').split(',').filter(Boolean) : null,
  };
}

// ---------------------------------------------------------------------------
// States: what to show, how to get there, what to wait for.
// `run(page)` navigates, interacts and settles; returns { fullPage }.
// ---------------------------------------------------------------------------
const states = [
  {
    id: 'dashboard',
    run: async (page) => {
      await page.goto(`${base}/`, { waitUntil: 'load' });
      await page.getByText('External IP').waitFor();
      await page.waitForTimeout(600);
      return { fullPage: true };
    },
  },
  {
    id: 'dashboard-portal', // mock in portal mode: banner + warning badge
    mode: 'portal',
    run: async (page) => {
      await page.goto(`${base}/`, { waitUntil: 'load' });
      await page.getByText('Captive portal detected').waitFor();
      await page.waitForTimeout(600);
      return { fullPage: true };
    },
  },
  {
    id: 'dashboard-connect', // profile-picker modal opened from a disconnected Wi-Fi card
    run: async (page) => {
      await page.goto(`${base}/`, { waitUntil: 'load' });
      await page.getByText('External IP').waitFor();
      const wifiSection = page.locator('section').filter({ hasText: 'Wi-Fi Interfaces' });
      await wifiSection.getByRole('button', { name: 'Connect', exact: true }).click();
      await page.locator('.modal-open').waitFor();
      await page.getByText('Café_Central', { exact: true }).waitFor();
      await page.waitForTimeout(400);
      return { fullPage: false };
    },
  },
  {
    id: 'wifi',
    run: async (page) => {
      await page.goto(`${base}/wifi?iface=wlan0`, { waitUntil: 'load' });
      await page.getByText('Café_Central').waitFor();
      await page.waitForTimeout(500);
      return { fullPage: true };
    },
  },
  {
    id: 'wifi-join',
    run: async (page) => {
      await page.goto(`${base}/wifi?iface=wlan0`, { waitUntil: 'load' });
      await page.getByText('Café_Central').waitFor();
      const row = page.locator('tr').filter({ hasText: 'Café_Central' });
      await row.getByRole('button', { name: 'Join' }).click();
      await page.getByText(/Connect to "Café_Central"/).waitFor();
      await page.locator('input[placeholder="Password"]').fill('s3cret-pass');
      await page.waitForTimeout(400);
      return { fullPage: false };
    },
  },
  {
    id: 'portal',
    run: async (page) => {
      const url = encodeURIComponent('http://10.0.0.1/login');
      await page.goto(`${base}/portal?url=${url}`, { waitUntil: 'load' });
      await page.locator('iframe[title="Portal"]').waitFor();
      await page.frameLocator('iframe[title="Portal"]').getByText('Hotel Central').waitFor({ timeout: 5000 });
      await page.waitForTimeout(700);
      return { fullPage: true };
    },
  },
  {
    id: 'portal-open-tab', // risk-acknowledgement dialog for opening outside the sandbox
    run: async (page) => {
      const url = encodeURIComponent('http://10.0.0.1/login');
      await page.goto(`${base}/portal?url=${url}`, { waitUntil: 'load' });
      await page.locator('iframe[title="Portal"]').waitFor();
      await page.frameLocator('iframe[title="Portal"]').getByText('Hotel Central').waitFor({ timeout: 5000 });
      await page.waitForTimeout(500);
      await page.getByTitle('Open through the proxy in a new tab').click();
      await page.getByText('Open in a new tab?').waitFor();
      await page.waitForTimeout(400);
      return { fullPage: false };
    },
  },
  {
    id: 'devices',
    run: async (page) => {
      await page.goto(`${base}/devices`, { waitUntil: 'load' });
      await page.getByText('wwan0').waitFor();
      await page.waitForTimeout(500);
      return { fullPage: true };
    },
  },
  {
    id: 'profiles',
    run: async (page) => {
      await page.goto(`${base}/connections`, { waitUntil: 'load' });
      await page.getByText('MyVPN').waitFor();
      await page.waitForTimeout(500);
      return { fullPage: true };
    },
  },
  {
    id: 'profile-new',
    run: async (page) => {
      await page.goto(`${base}/connections`, { waitUntil: 'load' });
      await page.getByText('MyVPN').waitFor();
      await page.getByRole('button', { name: 'New Profile' }).click();
      await page.locator('.modal-open').waitFor();
      await page.waitForTimeout(500);
      return { fullPage: false };
    },
  },
  {
    id: 'power',
    run: async (page) => {
      await page.goto(`${base}/power`, { waitUntil: 'load' });
      await page.locator('input[placeholder="Power password"]').waitFor();
      await page.locator('input[placeholder="Power password"]').fill('admin');
      await page.waitForTimeout(400);
      return { fullPage: true };
    },
  },
  {
    id: 'power-confirm',
    run: async (page) => {
      await page.goto(`${base}/power`, { waitUntil: 'load' });
      await page.locator('input[placeholder="Power password"]').fill('admin');
      await page.getByRole('button', { name: 'Reboot' }).first().click();
      await page.getByText('I understand the server will become unavailable.').waitFor();
      await page.waitForTimeout(400);
      return { fullPage: false };
    },
  },
  {
    id: 'nav-open', // mobile-only: hamburger menu
    run: async (page) => {
      await page.goto(`${base}/`, { waitUntil: 'load' });
      await page.getByText('External IP').waitFor();
      await page.getByRole('button', { name: 'Menu' }).click();
      await page.waitForTimeout(400);
      return { fullPage: false };
    },
  },
];

let base = 'http://127.0.0.1:18080'; // patched in by captureAll via closure? see below
const fileFor = (state, theme, vp) =>
  `${state}${theme === 'dark' ? '-dark' : ''}${vp === 'mobile' ? '-mobile' : ''}.png`;

// setMockMode switches the mock backend between serving 'online' and 'portal'
// data (see mock-server.mjs). Every state declares the data set it needs via
// `mode`; states without one default to 'online' so they never inherit the
// portal banner from an earlier capture.
async function setMockMode(mode) {
  const res = await fetch(`${base}/__mock/mode`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ mode }),
  });
  if (!res.ok) {
    throw new Error(`[screenshots] failed to set mock mode ${mode}: HTTP ${res.status}`);
  }
  return mode;
}

export async function captureAll({ baseUrl, outDir }) {
  base = baseUrl;
  mkdirSync(outDir, { recursive: true });
  const args = parseArgs();
  const onlyStates = args.states || null;

  const browser = await launchBrowser();
  const saved = [];
  try {
    for (const theme of ['light', 'dark']) {
      const context = await browser.newContext({ viewport: DESKTOP, colorScheme: theme });
      await context.addInitScript((t) => localStorage.setItem('nm-webui-theme', t), theme);
      for (const vp of ['desktop', 'mobile']) {
        const page = await context.newPage();
        await page.setViewportSize(vp === 'mobile' ? MOBILE : DESKTOP);
        for (const state of states) {
          if (onlyStates && !onlyStates.includes(state.id)) continue;
          if (state.id === 'nav-open' && vp === 'desktop') continue; // mobile-only state
          await setMockMode(state.mode || 'online');
          const { fullPage } = await state.run(page);
          const file = `${outDir}/${fileFor(state.id, theme, vp)}`;
          await page.screenshot({ path: file, fullPage });
          saved.push(file);
          console.log(`[screenshots] saved ${file}`);
        }
        await page.close();
      }
      await context.close();
    }
  } finally {
    await browser.close();
  }
  console.log(`[screenshots] DONE ${saved.length} screenshots -> ${outDir}`);
  return saved;
}

async function launchBrowser() {
  const candidates = [
    process.env.CHROMIUM_PATH,
    '/usr/bin/chromium',
    '/usr/bin/google-chrome-stable',
    '/usr/bin/chromium-browser',
    '/usr/local/bin/chromium',
    '/snap/bin/chromium',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  ].filter(Boolean);

  const errors = [];
  for (const exe of candidates) {
    try {
      return await chromium.launch({
        executablePath: exe,
        headless: true,
        args: ['--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage'],
      });
    } catch (e) {
      errors.push(`${exe}: ${e.message.split('\n')[0]}`);
    }
  }
  try {
    return await chromium.launch({ headless: true });
  } catch (e) {
    errors.push(`playwright chromium: ${e.message.split('\n')[0]}`);
  }
  throw new Error(
    'Could not launch any Chromium for screenshots.\n' +
      'Install one, e.g.:\n' +
      '  npx playwright install chromium\n' +
      'or point CHROMIUM_PATH at an existing chrome/chromium binary.\n' +
      'Attempted:\n  ' + errors.join('\n  ')
  );
}

// Standalone mode: node capture.mjs --base ... --out ...
const isMain = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;
if (isMain) {
  const args = parseArgs();
  await captureAll({ baseUrl: args.base, outDir: args.out });
}