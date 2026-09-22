# Documentation screenshots

Regenerates [`docs/screenshots/`](../screenshots/) (referenced by
[`docs/SCREENSHOTS.md`](../SCREENSHOTS.md)) against a **mock backend** with
fictitious "travel router" data — no NetworkManager, no real network devices,
nothing is ever touched on the host.

## How to regenerate (after frontend changes)

```bash
make screenshots
```

That single command:

1. builds the frontend (`cd webui && npm ci && npm run build`),
2. starts the zero-dependency mock HTTP server (ephemeral port),
3. renders every state with headless Chromium via Playwright in
   light + dark themes × desktop (1280×800) + mobile (390×844) viewports,
4. overwrites the PNGs in `docs/screenshots/` (file names are stable, so
   links in `docs/SCREENSHOTS.md` never change),
5. validates that every screenshot referenced by `docs/SCREENSHOTS.md` exists.

Afterwards review the visual diff with `git status` / `git diff --stat docs/`.

## Layout

| File | Purpose |
|---|---|
| `run.mjs` | orchestrator: build-check → mock → capture → link check |
| `mock-server.mjs` | `node:http` backend: serves the built SPA + `/api/v1/*` + SSE + fake portal page |
| `mock-data.mjs` | the test dataset — **edit this to change what screenshots show** |
| `capture.mjs` | Playwright capture: states × themes × viewports |
| `check-links.mjs` | verifies every `screenshots/…` path in `docs/SCREENSHOTS.md` resolves |
| `package.json` | dev dependency: `playwright` |

## Captured states

`dashboard` · `wifi` · `wifi-join` (password modal) · `portal` (mini-browser with
a fake hotel sign-in page) · `portal-open-tab` (risk-acknowledgement dialog for
opening a page outside the sandbox) · `devices` · `profiles` · `profile-new`
(profile dialog) · `power` · `power-confirm` (confirmation dialog) · `nav-open`
(mobile nav menu).

Each state → up to 4 files:

```
<state>.png            light / desktop
<state>-dark.png       dark  / desktop
<state>-mobile.png     light / mobile
<state>-dark-mobile.png dark  / mobile
```

## Prerequisites

- Node.js ≥ 18 (ES modules + top-level await; Playwright 1.63 needs ≥ 18).
- A Chromium binary. `capture.mjs` resolves it in this order:
  1. `CHROMIUM_PATH` env var,
  2. common system locations (`/usr/bin/chromium`, `google-chrome-stable`, …),
  3. Playwright's bundled browser — install with
     `cd scripts/screenshots && npm ci && npx playwright install chromium`.

## Fine-grained usage

```bash
# One-command run (works from anywhere in the repo)
node scripts/screenshots/run.mjs

# Reuse an existing build; only restart the server + capture
cd scripts/screenshots && npm ci
node run.mjs

# Capture a subset while iterating on the frontend
node capture.mjs --base http://127.0.0.1:18080 --out ../docs/screenshots --states dashboard,wifi

# Point at a different frontend build / output dir
NM_WEBUI_DIST=/custom/dist OUT=/tmp/shots node run.mjs

# Check links only
node check-links.mjs
```

## When a capture breaks

If the frontend changes a selector, text or route that a state waits on, that
state throws with the failing step instead of writing a wrong screenshot —
look at `capture.mjs` (the `states` table) and update the wait/selector to
match, or add a new state next to the existing ones.

## Editing the dataset

New/changed data lives in `mock-data.mjs`. `SCREENSHOTS.md` is updated by hand
when sections are added or removed; the index table and `<picture>`/`<details>`
blocks are plain Markdown.