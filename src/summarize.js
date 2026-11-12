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

export async function summarize(pr, { model = 'claude-opus-5-5', client = new Anthropic() } = {}) {
  const res = await client.beta.messages.parse({
    model,
    max_tokens: 16000,
    betas: ['server-side-fallback-2026-07-01'],
    fallbacks: 'default',
    system: SYSTEM,
    messages: [{ role: 'user', content: promptFor(pr) }],
    output_config: { format: betaZodOutputFormat(Notes) },
  });
  if (res.stop_reason === 'refusal') throw new Error(`Modelo recusou: ${res.stop_details?.category ?? 'sem categoria'}`);
  if (!res.parsed_output) throw new Error(`Resposta fora do formato (stop_reason=${res.stop_reason})`);
  return res.parsed_output;
}
