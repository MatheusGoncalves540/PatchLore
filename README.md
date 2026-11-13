# PatchLore

Patchnotes visuais e padronizados a partir do histórico do git.

A IA **só escreve o texto**, num JSON de formato fixo (`titulo` + até 6 `itens`, cada um com uma tag do tema).
Quem desenha é o próprio PatchLore, sempre com o mesmo layout, então todo patchnotes sai com a mesma cara.

```
commits e arquivos entre duas refs → IA (JSON validado) → layout fixo → PNG + PDF + Markdown
```

É um executável único, sem Node e sem navegador. Funciona com Claude, DeepSeek ou qualquer API compatível com OpenAI (OpenRouter, Groq, Ollama, LM Studio…).

## Instalar

Com Go 1.26 ou mais novo:

```bash
go install github.com/MatheusGoncalves540/patchlore@latest
```

Também é preciso ter o `git` no PATH.

## Configurar a IA

```bash
patchlore config
```

Abre um menu para adicionar, editar, remover e escolher o provedor padrão.
Claude e DeepSeek já vêm com URL e modelo preenchidos; basta colar a chave.
A lista de modelos é buscada **ao vivo** na API do provedor, então modelos novos aparecem sem atualizar o PatchLore. Se a busca falhar, você digita o nome.

Para outra API, escolha **Adicionar outro** e informe nome, formato (OpenAI ou Anthropic), URL base, chave e modelo.

Tudo fica em `~/.config/patchlore/config.toml` (ou em `$XDG_CONFIG_HOME/patchlore`), com permissão `600`.
As chaves ficam só nesse arquivo, nunca no repositório.

### Variáveis de ambiente

Fora do terminal (CI), estas variáveis substituem a config salva:

| Variável | O que faz |
|---|---|
| `PATCHLORE_API_KEY` | Chave da API. Se estiver definida, a config salva é ignorada |
| `PATCHLORE_PROVIDER` | Preset: `claude` (padrão) ou `deepseek` |
| `PATCHLORE_MODEL` | Modelo (vazio = o do preset) |
| `PATCHLORE_BASE_URL` | URL base de outra API |
| `PATCHLORE_API_TYPE` | Formato dessa API: `anthropic` ou `openai` |

## Gerar

Dentro do repositório, diga de onde até onde:

```bash
patchlore production..development
```

O PatchLore lê os commits que entraram em `development` desde `production` (sem os merges) e os arquivos alterados.
Não roda `git fetch`; use as refs como estão na sua máquina (`origin/main`, tags e branches locais valem).

As refs ficam salvas no `.patchlore.toml` da raiz do repo. Da próxima vez, basta:

```bash
patchlore
```

Sem refs salvas, `patchlore` abre um menu com as branches locais, remotas e tags.

| Opção | O que faz |
|---|---|
| `-o pasta` | Pasta de saída (padrão `patchlore-out`) |
| `-t "título"` | Título do rodapé, também enviado à IA (padrão: `antiga → nova`) |
| `-p nome` | Provedor salvo (padrão: o marcado com ★) |
| `-m modelo` | Troca o modelo só nesta execução |
| `-n notes.json` | Usa um JSON pronto em vez de chamar a IA |
| `--open` | Abre o PNG ao terminar |

Gera `notes.json`, `patchnotes.png`, `patchnotes.pdf` e `patchnotes.md`.
Quer corrigir o texto? Edite `notes.json` e rode de novo com `-n`; a IA não é chamada.

## Mudar o visual

Cores, tags (rótulo, cor, ícone), limite de itens e o prompt da IA vêm de um tema.
O tema padrão está em [`internal/tema/tema.toml`](internal/tema/tema.toml); copie só o que quiser mudar para:

- `~/.config/patchlore/tema.toml`: vale para todos os seus repositórios;
- a seção `[tema]` do `.patchlore.toml`: vale só para este repo e vence a do usuário.

Exemplo de `.patchlore.toml`:

```toml
antiga = "production"
nova = "development"

[tema.cores]
fundo = "#ffffff"

[tema.tags.novo]
fundo = "#16a34a"
cor = "#16a34a"

# Uma tag nova também entra no que a IA pode usar.
[tema.tags.seguranca]
rotulo = "SEGURANÇA"
descricao = "correção de segurança"
cor = "#b91c1c"
texto = "#ffffff"
fundo = "#b91c1c"
icone = "M12 3l8 4v5c0 5-3.5 8-8 9-4.5-1-8-4-8-9V7z"
```

As cores são `#rrggbb` e o ícone é um caminho SVG (o atributo `d`) numa caixa de 24×24, desenhado só com traço.
Não é preciso recompilar: a próxima execução já usa o tema novo.

## Modo PR e GitHub Action

A GitHub Action e o modo PR (`--pr`, `--comment`) ainda usam a versão em Node (`src/`) enquanto a migração para o executável Go não termina.

## Desenvolvimento

```bash
go test ./...
go build -o patchlore .
```

A fonte Inter vem embutida sob a [SIL Open Font License](internal/render/fontes/OFL.txt).
