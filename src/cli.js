#!/usr/bin/env node
import { mkdir, writeFile, readFile } from 'node:fs/promises';
import { parseArgs, styleText } from 'node:util';
import { join, resolve } from 'node:path';
import { spawn } from 'node:child_process';
import * as p from '@clack/prompts';
import { fetchPR, uploadImage, upsertComment, githubToken, detectRepo, currentBranch, findPRForBranch } from './github.js';
import { summarize, Notes } from './summarize.js';
import { renderHtml, renderMarkdown, TAG } from './render.js';
import { loadConfig, resolveProvider } from './config.js';
import { configMenu } from './configure.js';

const HELP = `${styleText('bold', 'patchlore')}: patchnotes visuais a partir de uma PR

${styleText('bold', 'Uso')}
  patchlore [pr]            gera o patchnotes (detecta repo e PR da branch atual)
  patchlore config          configura provedores de IA (Claude, DeepSeek, outros)

${styleText('bold', 'Opções')}
  -r, --repo dono/repo      repositório (padrão: remote origin)
  -p, --provider nome       provedor salvo (padrão: o marcado como padrão)
  -m, --model id            troca o modelo só nesta execução
  -o, --out dir             pasta de saída (padrão: patchlore-out)
  -n, --notes notes.json    usa um JSON pronto em vez de chamar a IA
  -c, --comment             publica a imagem e comenta na PR
      --open                abre o PNG ao terminar
  -h, --help                esta ajuda`;

const { values: a, positionals } = parseArgs({
  allowPositionals: true,
  options: {
    repo: { type: 'string', short: 'r' },
    pr: { type: 'string' },
    provider: { type: 'string', short: 'p' },
    model: { type: 'string', short: 'm' },
    out: { type: 'string', short: 'o', default: 'patchlore-out' },
    notes: { type: 'string', short: 'n' },
    comment: { type: 'boolean', short: 'c', default: false },
    open: { type: 'boolean', default: false },
    'assets-branch': { type: 'string', default: 'patchlore-assets' },
    help: { type: 'boolean', short: 'h', default: false },
  },
});

const fail = (msg) => {
  p.cancel(msg);
  process.exit(1);
};

if (a.help) {
  console.log(HELP);
  process.exit(0);
}
if (positionals[0] === 'config') {
  await configMenu();
  process.exit(0);
}

const interactive = process.stdout.isTTY && !p.isCI();
p.intro(styleText(['bgBlue', 'white', 'bold'], ' patchlore '));

const token = githubToken();
const repo = a.repo ?? detectRepo();
if (!repo) fail('Não achei o repositório. Use --repo dono/repo.');

let number = a.pr ?? positionals[0];
if (!number) {
  const branch = currentBranch();
  number = branch && (await findPRForBranch(repo, branch, token).catch(() => null));
  if (!number && interactive) {
    const v = await p.text({ message: `Número da PR em ${repo}`, validate: (v) => (/^\d+$/.test(v ?? '') ? undefined : 'Só números') });
    if (p.isCancel(v)) fail('Cancelado.');
    number = v;
  }
  if (!number) fail('Não achei PR para a branch atual. Passe o número: patchlore 74');
}

const s = p.spinner();
try {
  s.start(`Lendo ${repo}#${number}`);
  const pr = await fetchPR(repo, number, token);
  s.stop(`${styleText('bold', pr.title)} ${styleText('dim', `· ${pr.commits.length} commits · ${pr.files.length} arquivos`)}`);

  let notes;
  if (a.notes) {
    notes = Notes.parse(JSON.parse(await readFile(a.notes, 'utf8')));
    p.log.info(`Usando ${a.notes}`);
  } else {
    const provider = resolveProvider(await loadConfig(), a);
    s.start(`Resumindo com ${provider.name} · ${provider.model}`);
    notes = await summarize(pr, provider);
    s.stop(`Resumo pronto (${provider.name} · ${provider.model})`);
  }

  s.start('Montando a imagem');
  await mkdir(a.out, { recursive: true });
  const html = renderHtml(notes, pr);
  const md = renderMarkdown(notes, pr);
  await writeFile(join(a.out, 'notes.json'), JSON.stringify(notes, null, 2));
  await writeFile(join(a.out, 'patchnotes.html'), html);
  await writeFile(join(a.out, 'patchnotes.md'), md);
  const png = await screenshot(html);
  const pngPath = join(a.out, 'patchnotes.png');
  await writeFile(pngPath, png);
  s.stop('Imagem pronta');

  p.note(
    notes.itens.map((it) => `${styleText('bold', TAG[it.tag].label.padEnd(9))} ${it.titulo}`).join('\n'),
    notes.titulo,
  );

  if (a.comment) {
    s.start('Comentando na PR');
    const img = await uploadImage(repo, a['assets-branch'], `pr-${pr.number}.png`, png, token);
    await upsertComment(repo, pr.number, `![Patchnotes](${img})\n\n<details><summary>Texto</summary>\n\n${md}\n</details>`, token);
    s.stop(`Comentado em ${pr.url}`);
  }
  if (a.open) openFile(resolve(pngPath));

  p.outro(`${styleText('green', '✔')} ${resolve(a.out)}  ${styleText('dim', '(edite notes.json e rode com --notes para ajustar)')}`);
} catch (e) {
  s.stop(styleText('red', 'Falhou'), 1);
  fail(e.message);
}

async function screenshot(html) {
  const { chromium } = await import('playwright');
  const browser = await chromium.launch().catch(() => {
    throw new Error('Chromium não encontrado. Rode: npx playwright install chromium');
  });
  try {
    const page = await browser.newPage({ deviceScaleFactor: 2 });
    await page.setContent(html);
    return await page.locator('#pn').screenshot();
  } finally {
    await browser.close();
  }
}

function openFile(path) {
  const [cmd, args] =
    process.platform === 'darwin' ? ['open', [path]] : process.platform === 'win32' ? ['cmd', ['/c', 'start', '', path]] : ['xdg-open', [path]];
  spawn(cmd, args, { detached: true, stdio: 'ignore' }).unref();
}
