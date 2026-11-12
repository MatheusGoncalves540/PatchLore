import { test } from 'node:test';
import assert from 'node:assert/strict';
import { renderHtml, renderMarkdown } from '../src/render.js';
import { Notes, promptFor } from '../src/summarize.js';

const pr = {
  repo: 'dono/app', number: 7, title: 'Busca', body: '', head: 'dev', base: 'main',
  commits: ['feat: busca'], files: [{ name: 'a.ts', status: 'modified', additions: 1, deletions: 0 }],
};
const notes = { titulo: 'Busca <nova>', itens: [{ tag: 'regra', titulo: 'Sem "tipo"', texto: 'a & b' }] };

test('html escapa o texto da IA e mostra a tag', () => {
  const html = renderHtml(notes, pr);
  assert.match(html, /Busca &lt;nova&gt;/);
  assert.match(html, /Sem &quot;tipo&quot;/);
  assert.match(html, /a &amp; b/);
  assert.match(html, />REGRA</);
  assert.doesNotMatch(html, /<nova>/);
});

test('markdown traz itens e contagens', () => {
  assert.equal(renderMarkdown(notes, pr).includes('**REGRA · Sem "tipo"**: a & b'), true);
  assert.match(renderMarkdown(notes, pr), /1 commits · 1 arquivos/);
});

test('schema recusa tag desconhecida', () => {
  assert.throws(() => Notes.parse({ titulo: 'x', itens: [{ tag: 'feature', titulo: 'a', texto: 'b' }] }));
});

test('prompt inclui commits e arquivos', () => {
  const p = promptFor(pr);
  assert.match(p, /- feat: busca/);
  assert.match(p, /modified \+1 -0 a\.ts/);
});
