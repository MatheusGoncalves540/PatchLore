# PatchLore

Patchnotes visuais e padronizados para cada PR.

A IA **só escreve o texto**, num JSON de formato fixo (`titulo` + até 6 `itens` com tag `novo`, `melhoria`, `correcao` ou `regra`).
Quem monta a tela é um template HTML fixo, então todo patchnotes sai com o mesmo visual.

```
PR (título, descrição, commits, arquivos) → IA (JSON validado) → template → PNG + Markdown → comentário na PR
```

Funciona com Claude, DeepSeek ou qualquer API compatível com OpenAI (OpenRouter, Groq, Ollama, LM Studio…).

## Instalar

```bash
npm install -g github:MatheusGoncalves540/PatchLore
npx playwright install chromium
```

## Configurar a IA

```bash
patchlore config
```

Menu interativo para adicionar, editar, remover e escolher o provedor padrão.
Claude e DeepSeek já vêm com URL e modelo preenchidos; basta colar a chave.
A lista de modelos é buscada **ao vivo** na API do provedor (`/v1/models`), então modelos novos aparecem sem atualizar o PatchLore. Se a busca falhar, você digita o nome.

Para outra API, escolha **Adicionar outro** e informe nome, formato (OpenAI ou Anthropic), URL base, chave e modelo.

A config fica em `~/.config/patchlore/config.json`, com permissão `600` (só seu usuário lê).

## Gerar

Dentro do repositório, na branch da PR:

```bash
patchlore
```

O repositório vem do `remote origin` e a PR, da branch atual. Também dá para ser explícito:

```bash
patchlore 74 -r dono/repo
```

| Opção | O que faz |
|---|---|
| `-r, --repo dono/repo` | Repositório (padrão: remote origin) |
| `-p, --provider nome` | Provedor salvo (padrão: o marcado com ★) |
| `-m, --model id` | Troca o modelo só nesta execução |
| `-o, --out dir` | Pasta de saída (padrão `patchlore-out`) |
| `-n, --notes notes.json` | Usa um JSON pronto em vez de chamar a IA |
| `-c, --comment` | Sobe a imagem e comenta na PR |
| `--open` | Abre o PNG ao terminar |

Gera `notes.json`, `patchnotes.html`, `patchnotes.png` e `patchnotes.md`.
Quer corrigir o texto? Edite `notes.json` e rode de novo com `--notes`.

O token do GitHub vem de `GITHUB_TOKEN`, `GH_TOKEN` ou do `gh auth token`.

## GitHub Action

1. Crie o secret com a chave da IA no repositório (Settings → Secrets and variables → Actions).
2. Adicione `.github/workflows/patchlore.yml`:

```yaml
name: PatchLore
on:
  pull_request:
    types: [opened, ready_for_review]
permissions:
  contents: write       # guarda a imagem na branch patchlore-assets
  pull-requests: write  # comenta na PR
jobs:
  patchnotes:
    if: ${{ !github.event.pull_request.draft }}
    runs-on: ubuntu-latest
    steps:
      - uses: MatheusGoncalves540/PatchLore@main
        with:
          api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          # provider: deepseek        # ou claude (padrão)
          # model: deepseek-reasoner  # opcional
          # base-url: https://openrouter.ai/api/v1   # outra API…
          # api-type: openai                          # …e seu formato
```

O comentário é editado no lugar a cada execução, sem duplicar. Para gerar de novo depois de novos commits, adicione `synchronize` em `types` (cada execução é uma chamada paga à API).

Se o PatchLore for privado, libere o uso em **Settings → Actions → General → Access** dele.

Fora do terminal (CI), as variáveis `PATCHLORE_API_KEY`, `PATCHLORE_PROVIDER`, `PATCHLORE_MODEL`, `PATCHLORE_BASE_URL` e `PATCHLORE_API_TYPE` substituem a config salva.

## Testes

```bash
npm test
```
