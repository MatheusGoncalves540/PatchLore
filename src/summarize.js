import Anthropic from '@anthropic-ai/sdk';
import { betaZodOutputFormat } from '@anthropic-ai/sdk/helpers/beta/zod';
import { z } from 'zod';

export const TAGS = ['novo', 'melhoria', 'correcao', 'regra'];

export const Notes = z.object({
  titulo: z.string().describe('Frase curta que resume a entrega, sem ponto final'),
  itens: z
    .array(
      z.object({
        tag: z.enum(TAGS),
        titulo: z.string().describe('2 a 4 palavras'),
        texto: z.string().describe('1 ou 2 frases, até ~240 caracteres, do ponto de vista de quem usa o sistema'),
      }),
    )
    .min(1)
    .max(6)
    .describe('De 1 a 6 itens, do mais importante para o menos'),
});

const SYSTEM = `Você escreve patchnotes para quem USA o sistema, não para devs.
Leia o histórico da PR e devolva só o que muda na experiência do usuário.
Ignore testes, refactors internos, docs técnicas, planos, CI, dependências e merges.
Agrupe mudanças relacionadas num item só. Nada de jargão de código, nomes de arquivo ou de função.
Escreva em português do Brasil, frases diretas. Não invente nada que o histórico não mostre.
Tags: novo = funcionalidade nova; melhoria = algo existente ficou melhor; correcao = bug corrigido; regra = nova validação ou restrição.`;

export function promptFor(pr) {
  const files = pr.files.map((f) => `${f.status} +${f.additions} -${f.deletions} ${f.name}`).join('\n');
  return [
    `# PR #${pr.number}: ${pr.title}`,
    `${pr.head} → ${pr.base}`,
    `## Descrição\n${pr.body || '(vazia)'}`,
    `## Commits\n${pr.commits.map((m) => `- ${m}`).join('\n')}`,
    `## Arquivos\n${files}`,
  ].join('\n\n');
}

const trimSlash = (u) => u.replace(/\/+$/, '');
const isOfficialAnthropic = (p) => new URL(p.baseUrl).hostname === 'api.anthropic.com';

function anthropic(p) {
  return new Anthropic({ apiKey: p.apiKey, baseURL: trimSlash(p.baseUrl) });
}

async function http(url, apiKey, body) {
  const res = await fetch(url, {
    method: body ? 'POST' : 'GET',
    headers: { authorization: `Bearer ${apiKey}`, 'content-type': 'application/json' },
    body: body && JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`${res.status} em ${url}: ${(await res.text()).slice(0, 300)}`);
  return res.json();
}

// Lista ao vivo os modelos do provedor, para não depender de lista fixa.
export async function listModels(p) {
  if (p.type === 'anthropic') {
    const ids = [];
    for await (const m of anthropic(p).models.list()) ids.push(m.id);
    return ids;
  }
  const { data } = await http(`${trimSlash(p.baseUrl)}/models`, p.apiKey);
  return data.map((m) => m.id).sort();
}

async function viaAnthropic(p, pr) {
  const official = isOfficialAnthropic(p);
  const res = await anthropic(p).beta.messages.parse({
    model: p.model,
    max_tokens: 16000,
    // Server-side fallback só existe na API oficial; proxies podem rejeitar o campo.
    ...(official && { betas: ['server-side-fallback-2026-07-01'], fallbacks: 'default' }),
    system: SYSTEM,
    messages: [{ role: 'user', content: promptFor(pr) }],
    output_config: { format: betaZodOutputFormat(Notes) },
  });
  if (res.stop_reason === 'refusal') throw new Error(`Modelo recusou: ${res.stop_details?.category ?? 'sem categoria'}`);
  if (!res.parsed_output) throw new Error(`Resposta fora do formato (stop_reason=${res.stop_reason})`);
  return res.parsed_output;
}

async function viaOpenAI(p, pr) {
  const schema = JSON.stringify(z.toJSONSchema(Notes));
  const res = await http(`${trimSlash(p.baseUrl)}/chat/completions`, p.apiKey, {
    model: p.model,
    max_tokens: 8000,
    response_format: { type: 'json_object' },
    messages: [
      { role: 'system', content: `${SYSTEM}\n\nResponda APENAS com um JSON válido neste JSON Schema:\n${schema}` },
      { role: 'user', content: promptFor(pr) },
    ],
  });
  const text = res.choices?.[0]?.message?.content;
  if (!text) throw new Error(`Resposta vazia (finish_reason=${res.choices?.[0]?.finish_reason})`);
  return parseNotes(text);
}

// Valida JSON vindo de modelo sem saída estruturada nativa; aceita cerca ```json.
export function parseNotes(text) {
  const json = text.trim().replace(/^```(?:json)?\s*|\s*```$/g, '');
  const result = Notes.safeParse(JSON.parse(json));
  if (!result.success) throw new Error(`JSON fora do formato: ${result.error.issues.map((i) => i.path.join('.') + ' ' + i.message).join('; ')}`);
  return result.data;
}

export function summarize(pr, provider) {
  return provider.type === 'anthropic' ? viaAnthropic(provider, pr) : viaOpenAI(provider, pr);
}
