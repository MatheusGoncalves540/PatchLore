import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { resolveProvider, PRESETS } from '../src/config.js';
import { parseRemote } from '../src/github.js';
import { parseNotes, listModels, summarize } from '../src/summarize.js';

const cfg = {
  default: 'deepseek',
  providers: { deepseek: { ...PRESETS.deepseek, apiKey: 'k1' }, meu: { name: 'meu', type: 'openai', baseUrl: 'http://x', model: 'm', apiKey: 'k2' } },
};

test('resolveProvider: padrão, --provider e --model', () => {
  assert.equal(resolveProvider(cfg, {}, {}).name, 'deepseek');
  assert.equal(resolveProvider(cfg, { provider: 'meu' }, {}).apiKey, 'k2');
  assert.equal(resolveProvider(cfg, { model: 'deepseek-reasoner' }, {}).model, 'deepseek-reasoner');
  assert.equal(cfg.providers.deepseek.model, 'deepseek-chat', 'não altera a config salva');
});

test('resolveProvider: env vence a config e usa o preset', () => {
  const p = resolveProvider(cfg, {}, { PATCHLORE_API_KEY: 'env', PATCHLORE_PROVIDER: 'claude' });
  assert.deepEqual([p.type, p.baseUrl, p.model, p.apiKey], ['anthropic', 'https://api.anthropic.com', 'claude-opus-5-5', 'env']);
});

test('resolveProvider: erros claros', () => {
  assert.throws(() => resolveProvider({ providers: {}, default: null }, {}, {}), /patchlore config/);
  assert.throws(() => resolveProvider(cfg, { provider: 'nada' }, {}), /não existe/);
});

test('parseRemote', () => {
  assert.equal(parseRemote('git@github.com:dono/repo.git'), 'dono/repo');
  assert.equal(parseRemote('https://github.com/dono/repo'), 'dono/repo');
  assert.equal(parseRemote('https://gitlab.com/dono/repo'), null);
});

test('parseNotes aceita cerca ```json e recusa formato errado', () => {
  const ok = parseNotes('```json\n{"titulo":"t","itens":[{"tag":"novo","titulo":"a","texto":"b"}]}\n```');
  assert.equal(ok.itens[0].tag, 'novo');
  assert.throws(() => parseNotes('{"titulo":"t","itens":[]}'), /fora do formato/);
});

test('API compatível com OpenAI: lista modelos e resume', async () => {
  const seen = [];
  const server = createServer((req, res) => {
    let body = '';
    req.on('data', (c) => (body += c));
    req.on('end', () => {
      seen.push({ url: req.url, auth: req.headers.authorization, body: body && JSON.parse(body) });
      res.setHeader('content-type', 'application/json');
      if (req.url === '/v1/models') return res.end(JSON.stringify({ data: [{ id: 'b' }, { id: 'a' }] }));
      const notes = { titulo: 't', itens: [{ tag: 'regra', titulo: 'x', texto: 'y' }] };
      res.end(JSON.stringify({ choices: [{ message: { content: JSON.stringify(notes) } }] }));
    });
  });
  await new Promise((r) => server.listen(0, r));
  const prov = { type: 'openai', baseUrl: `http://127.0.0.1:${server.address().port}/v1/`, model: 'a', apiKey: 'sk' };
  try {
    assert.deepEqual(await listModels(prov), ['a', 'b']);
    const pr = { number: 1, title: 'T', body: '', head: 'h', base: 'b', commits: ['c'], files: [] };
    assert.equal((await summarize(pr, prov)).itens[0].tag, 'regra');
    const chat = seen.find((s) => s.url === '/v1/chat/completions');
    assert.equal(chat.auth, 'Bearer sk');
    assert.equal(chat.body.response_format.type, 'json_object');
  } finally {
    server.close();
  }
});
