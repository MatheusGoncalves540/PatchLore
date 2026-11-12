import { styleText } from 'node:util';
import * as p from '@clack/prompts';
import { PRESETS, CONFIG_PATH, loadConfig, saveConfig } from './config.js';
import { listModels } from './summarize.js';

const OTHER = Symbol('outro');

function ask(v) {
  if (p.isCancel(v)) {
    p.cancel('Nada foi salvo.');
    process.exit(0);
  }
  return v;
}

const maskKey = (k) => (k ? `••••${k.slice(-4)}` : styleText('red', 'sem chave'));

function summary(cfg) {
  const names = Object.keys(cfg.providers);
  if (!names.length) return styleText('dim', 'Nenhum provedor ainda.');
  return names
    .map((n) => {
      const pr = cfg.providers[n];
      const star = n === cfg.default ? styleText('green', '★') : ' ';
      return `${star} ${styleText('bold', n.padEnd(12))} ${pr.model.padEnd(22)} ${styleText('dim', pr.baseUrl)}  ${maskKey(pr.apiKey)}`;
    })
    .join('\n');
}

async function pickModel(prov) {
  const s = p.spinner();
  s.start(`Buscando modelos em ${prov.baseUrl}`);
  let models = [];
  try {
    models = await listModels(prov);
    s.stop(`${models.length} modelos disponíveis`);
  } catch (e) {
    s.stop(styleText('yellow', `Não deu para listar os modelos (${e.message.slice(0, 80)}). Digite o nome.`), 1);
  }
  if (models.length) {
    const choice = ask(
      await p.autocomplete({
        message: 'Modelo (digite para filtrar)',
        options: [...models.map((m) => ({ value: m, label: m, hint: m === prov.model ? 'atual' : undefined })), { value: OTHER, label: 'Outro…' }],
        initialValue: models.includes(prov.model) ? prov.model : models[0],
        maxItems: 10,
      }),
    );
    if (choice !== OTHER) return choice;
  }
  return ask(await p.text({ message: 'Modelo', initialValue: prov.model ?? '', validate: (v) => (v?.trim() ? undefined : 'Obrigatório') })).trim();
}

async function editProvider(cfg, base, { editing = false } = {}) {
  const name = ask(
    await p.text({
      message: 'Nome',
      initialValue: base.name ?? '',
      validate: (v) => {
        if (!/^[\w.-]+$/.test(v ?? '')) return 'Use letras, números, - _ .';
        if (v !== base.name && cfg.providers[v]) return 'Já existe um provedor com esse nome';
      },
    }),
  );
  const type =
    base.type ??
    ask(
      await p.select({
        message: 'Formato da API',
        options: [
          { value: 'openai', label: 'Compatível com OpenAI', hint: '/chat/completions: DeepSeek, OpenRouter, Groq, Ollama…' },
          { value: 'anthropic', label: 'Anthropic', hint: '/v1/messages' },
        ],
      }),
    );
  const baseUrl = ask(
    await p.text({
      message: 'URL base da API',
      initialValue: base.baseUrl ?? '',
      validate: (v) => (URL.canParse(v ?? '') ? undefined : 'URL inválida'),
    }),
  ).replace(/\/+$/, '');
  const typedKey = ask(
    await p.password({
      message: editing && base.apiKey ? 'Chave da API (Enter mantém a atual)' : 'Chave da API',
      validate: (v) => (v || (editing && base.apiKey) ? undefined : 'Obrigatória'),
    }),
  );
  const prov = { name, type, baseUrl, apiKey: typedKey || base.apiKey, model: base.model };
  prov.model = await pickModel(prov);

  if (editing && name !== base.name) {
    delete cfg.providers[base.name];
    if (cfg.default === base.name) cfg.default = name;
  }
  cfg.providers[name] = prov;
  if (!cfg.default) cfg.default = name;
  await saveConfig(cfg);
  p.log.success(`${name} salvo${cfg.default === name ? ' (padrão)' : ''}`);
}

async function chooseProvider(cfg, message) {
  return ask(await p.select({ message, options: Object.keys(cfg.providers).map((n) => ({ value: n, label: n, hint: cfg.providers[n].model })) }));
}

export async function configMenu() {
  p.intro(styleText(['bgBlue', 'white', 'bold'], ' patchlore config '));
  const cfg = await loadConfig();
  for (;;) {
    p.note(summary(cfg), 'Provedores');
    const has = Object.keys(cfg.providers).length > 0;
    const action = ask(
      await p.select({
        message: 'O que fazer?',
        options: [
          { value: 'claude', label: 'Adicionar Claude', hint: 'api.anthropic.com' },
          { value: 'deepseek', label: 'Adicionar DeepSeek', hint: 'api.deepseek.com' },
          { value: 'custom', label: 'Adicionar outro', hint: 'qualquer URL' },
          ...(has
            ? [
                { value: 'edit', label: 'Editar (URL, modelo, nome, chave)' },
                { value: 'default', label: 'Definir padrão' },
                { value: 'remove', label: 'Remover' },
              ]
            : []),
          { value: 'exit', label: 'Sair' },
        ],
      }),
    );
    if (action === 'exit') break;
    if (action === 'claude' || action === 'deepseek') {
      const preset = PRESETS[action];
      const free = cfg.providers[preset.name] ? `${preset.name}-2` : preset.name;
      await editProvider(cfg, { ...preset, name: free });
    } else if (action === 'custom') {
      await editProvider(cfg, {});
    } else if (action === 'edit') {
      const n = await chooseProvider(cfg, 'Qual editar?');
      await editProvider(cfg, cfg.providers[n], { editing: true });
    } else if (action === 'default') {
      cfg.default = await chooseProvider(cfg, 'Qual será o padrão?');
      await saveConfig(cfg);
    } else if (action === 'remove') {
      const n = await chooseProvider(cfg, 'Qual remover?');
      if (ask(await p.confirm({ message: `Remover ${n}?`, initialValue: false }))) {
        delete cfg.providers[n];
        if (cfg.default === n) cfg.default = Object.keys(cfg.providers)[0] ?? null;
        await saveConfig(cfg);
      }
    }
  }
  p.outro(styleText('dim', `Salvo em ${CONFIG_PATH}`));
}
