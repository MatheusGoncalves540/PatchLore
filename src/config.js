import { mkdir, readFile, writeFile, chmod } from 'node:fs/promises';
import { homedir } from 'node:os';
import { join, dirname } from 'node:path';

// type: 'anthropic' = API da Anthropic; 'openai' = qualquer API compatível com /chat/completions.
export const PRESETS = {
  claude: { name: 'claude', type: 'anthropic', baseUrl: 'https://api.anthropic.com', model: 'claude-opus-5-5' },
  deepseek: { name: 'deepseek', type: 'openai', baseUrl: 'https://api.deepseek.com', model: 'deepseek-chat' },
};

export const CONFIG_PATH = join(process.env.XDG_CONFIG_HOME || join(homedir(), '.config'), 'patchlore', 'config.json');

export async function loadConfig() {
  try {
    return JSON.parse(await readFile(CONFIG_PATH, 'utf8'));
  } catch (e) {
    if (e.code === 'ENOENT') return { providers: {}, default: null };
    throw new Error(`Config inválida em ${CONFIG_PATH}: ${e.message}`);
  }
}

export async function saveConfig(cfg) {
  await mkdir(dirname(CONFIG_PATH), { recursive: true, mode: 0o700 });
  await writeFile(CONFIG_PATH, JSON.stringify(cfg, null, 2), { mode: 0o600 });
  await chmod(CONFIG_PATH, 0o600); // the file may predate this run with looser permissions
}

// Env vars win (CI / GitHub Action); otherwise --provider, then the saved default.
export function resolveProvider(cfg, { provider, model } = {}, env = process.env) {
  let p;
  if (env.PATCHLORE_API_KEY) {
    const preset = PRESETS[env.PATCHLORE_PROVIDER || 'claude'] ?? {};
    p = {
      ...preset,
      name: env.PATCHLORE_PROVIDER || preset.name || 'env',
      type: env.PATCHLORE_API_TYPE || preset.type || 'openai',
      baseUrl: env.PATCHLORE_BASE_URL || preset.baseUrl,
      model: env.PATCHLORE_MODEL || preset.model,
      apiKey: env.PATCHLORE_API_KEY,
    };
  } else {
    const name = provider || cfg.default;
    if (!name) throw new Error('Nenhum provedor configurado. Rode `patchlore config`.');
    p = cfg.providers[name];
    if (!p) throw new Error(`Provedor "${name}" não existe. Rode \`patchlore config\`.`);
  }
  if (model) p = { ...p, model };
  if (!p.baseUrl || !p.model) throw new Error(`Provedor "${p.name}" sem URL ou modelo.`);
  return p;
}
