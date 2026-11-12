# PatchLore

Patchnotes visuais e padronizados para cada PR.

A IA **só escreve o texto**, num JSON de formato fixo (`titulo` + até 6 `itens` com tag `novo`, `melhoria`, `correcao` ou `regra`).
Quem monta a tela é um template HTML fixo, então todo patchnotes sai com o mesmo visual.

```
PR (título, descrição, commits, arquivos) → Claude (JSON validado) → template → PNG + Markdown → comentário na PR
```

## Usar em um repositório (GitHub Action)

1. No repositório, crie o secret `ANTHROPIC_API_KEY` (Settings → Secrets and variables → Actions).
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
          anthropic-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
```

O comentário é editado no lugar a cada execução, sem duplicar. Para gerar de novo depois de novos commits, adicione `synchronize` em `types` (cada execução é uma chamada paga à API).

Se o PatchLore for privado, libere o uso em **Settings → Actions → General → Access** dele ("Accessible from repositories owned by the user").

## Usar localmente (CLI)

```bash
npm install && npx playwright install chromium
GITHUB_TOKEN=... ANTHROPIC_API_KEY=... node src/cli.js --repo dono/repo --pr 74
```

Gera `patchlore-out/` com `notes.json`, `patchnotes.html`, `patchnotes.png` e `patchnotes.md`.

| Opção | O que faz |
|---|---|
| `--out dir` | Pasta de saída (padrão `patchlore-out`) |
| `--notes notes.json` | Reaproveita/edita um JSON em vez de chamar a IA |
| `--model id` | Modelo Claude (padrão `claude-opus-5-5`) |
| `--comment` | Sobe a imagem e comenta na PR |

Quer corrigir o texto? Edite `notes.json` e rode de novo com `--notes`.

## Testes

```bash
npm test
```
