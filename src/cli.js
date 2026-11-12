#!/usr/bin/env node
import { mkdir, writeFile, readFile } from 'node:fs/promises';
import { parseArgs } from 'node:util';
import { join } from 'node:path';
import { chromium } from 'playwright';
import { fetchPR, uploadImage, upsertComment } from './github.js';
import { summarize, Notes } from './summarize.js';
import { renderHtml, renderMarkdown } from './render.js';

const { values: a } = parseArgs({
  options: {
    repo: { type: 'string' },
    pr: { type: 'string' },
    out: { type: 'string', default: 'patchlore-out' },
    model: { type: 'string', default: 'claude-opus-5-5' },
    notes: { type: 'string' }, // reuse a notes.json instead of calling the AI
    comment: { type: 'boolean', default: false },
    'assets-branch': { type: 'string', default: 'patchlore-assets' },
  },
});
if (!a.repo || !a.pr) {
  console.error('uso: patchlore --repo dono/repo --pr 74 [--out dir] [--notes notes.json] [--comment]');
  process.exit(2);
}

const token = process.env.GITHUB_TOKEN;
const pr = await fetchPR(a.repo, a.pr, token);
const notes = a.notes ? Notes.parse(JSON.parse(await readFile(a.notes, 'utf8'))) : await summarize(pr, { model: a.model });

await mkdir(a.out, { recursive: true });
const html = renderHtml(notes, pr);
const md = renderMarkdown(notes, pr);
await writeFile(join(a.out, 'notes.json'), JSON.stringify(notes, null, 2));
await writeFile(join(a.out, 'patchnotes.html'), html);
await writeFile(join(a.out, 'patchnotes.md'), md);

const browser = await chromium.launch();
try {
  const page = await browser.newPage({ deviceScaleFactor: 2 });
  await page.setContent(html);
  const png = await page.locator('#pn').screenshot();
  await writeFile(join(a.out, 'patchnotes.png'), png);

  if (a.comment) {
    const img = await uploadImage(a.repo, a['assets-branch'], `pr-${pr.number}.png`, png, token);
    await upsertComment(a.repo, pr.number, `![Patchnotes](${img})\n\n<details><summary>Texto</summary>\n\n${md}\n</details>`, token);
  }
} finally {
  await browser.close();
}
console.log(`patchnotes em ${a.out}/`);
