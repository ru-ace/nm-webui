// Validate that every screenshots/ path referenced in docs/SCREENSHOTS.md
// points at an existing file. Exits non-zero when something is missing.
//
//   node check-links.mjs [docsDir]
import { readFile, access } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

export const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));

const REF_RE = /screenshots\/[\w-]+\.png/g;

export async function checkLinks(docsDir = join(REPO_ROOT, 'docs')) {
  const mdPath = join(docsDir, 'SCREENSHOTS.md');
  const md = await readFile(mdPath, 'utf8');
  const refs = [...new Set(md.match(REF_RE) || [])];
  const missing = [];
  for (const ref of refs) {
    try {
      await access(join(docsDir, ref));
    } catch {
      missing.push(ref);
    }
  }
  for (const r of refs) console.log(missing.includes(r) ? `MISSING ${r}` : `OK      ${r}`);
  if (missing.length) {
    console.error(`[screenshots] FAIL: ${missing.length} referenced file(s) missing from ${mdPath}`);
    process.exitCode = 1;
    return false;
  }
  console.log(`[screenshots] OK: all ${refs.length} screenshot links in ${mdPath} resolve`);
  return true;
}

const isMain = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;
if (isMain) {
  await checkLinks(process.argv[2] || join(REPO_ROOT, 'docs'));
}