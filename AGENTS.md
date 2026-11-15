# PatchLore — guia para agentes de código

Este arquivo é para quem (pessoa ou agente) vai **mexer no código e publicar versões**.
O `README.md` é para quem usa o programa; aqui ficam as regras que não podem quebrar.

## Contrato do projeto

1. **A IA só escreve texto.** Ela devolve um JSON validado (`titulo` + até 6 `itens`);
   quem desenha é `internal/render`. Nunca deixe o modelo escolher layout, cor ou ícone.
2. **O tema é a fonte da verdade.** Visual, tags, limite de itens e o prompt da IA vivem em
   `internal/tema/tema.toml`, com override em `~/.config/patchlore/tema.toml` e na seção
   `[tema]` do `.patchlore.toml`. Mudança de comportamento de texto começa pelo tema.
3. **O texto fala com o usuário final.** Só o que aparece na tela e no dia a dia; nada de
   arquivo, branch, commit, API, banco ou causa técnica. `internal/ia.Prompt` já filtra
   commits de manutenção e não envia caminhos de arquivo — não reintroduza isso.
4. **Nada de segredo no repositório.** Chave de API fica em `~/.config/patchlore/config.toml`
   (permissão `600`) ou em variável de ambiente.
5. **O repositório não ganha arquivos por causa de estado.** O checkpoint do
   `patchlore ultimo` fica em `~/.config/patchlore/estado.toml`, com uma entrada por repo.
   O `.patchlore.toml` continua sendo só do usuário, com as refs que ele escolheu.
6. **Tudo em português do Brasil:** comentários, mensagens de erro, interface e commits.

## Mapa do código

| Pacote | Papel |
|---|---|
| `main.go` | CLI, flags, subcomandos (`config`, `mcp`, `ultimo`), orquestração do fluxo |
| `mcp.go` | Servidor MCP (JSON-RPC por stdio) que expõe `gerar_patchnotes` |
| `internal/config` | Provedores/chaves e `.patchlore.toml` (refs do projeto) |
| `internal/estado` | Último commit processado por repositório, fora do repo |
| `internal/origem` | Lê commits/arquivos do git local (`Raiz`, `Refs`, `Resolve`, `Local`) |
| `internal/ia` | Prompt, schema e chamadas às APIs Anthropic/OpenAI |
| `internal/notas` | Tipos `Mudancas`/`Notas` e validação contra o tema |
| `internal/render` | PNG/PDF (canvas + Inter embutida) e Markdown |
| `internal/tema` | Tema embutido e merge das camadas |
| `internal/menu` | Menus interativos (refs e provedores) |

## Desenvolver e testar

```bash
gofmt -l .          # deve sair vazio
go vet ./...
go test ./...
go build -o patchlore.exe .
```

- O `git` precisa estar no `PATH`; os testes criam repositórios temporários de verdade.
- Testes de IA usam `httptest` e nunca chamam API real.
- Para testar o fluxo sem gastar token: `patchlore antiga..nova -n testdata/notes.json`.
- No Windows, `go install .` grava em `%USERPROFILE%\go\bin`; o `patchlore.exe` da raiz
  é ignorado pelo git e também pode estar no `PATH` (ele vence se vier primeiro).
- `gofmt -w .` reescreve o repo inteiro e deixa o `git status` barulhento por causa de
  CRLF. Formate só os arquivos que mexeu e confirme com `git diff --stat`.

## Publicar uma versão

Pré-requisitos: `gh` autenticado e acesso de escrita ao repositório.

1. Feche o código com `go test ./...` verde e commit em `main` (mensagem no padrão
   `tipo: resumo`, imperativo, até 50 caracteres).
2. Atualize a documentação de instalação se algo mudou no fluxo do usuário.
3. Crie e envie a tag anotada:

   ```bash
   git tag -a v0.2.2 -m "PatchLore v0.2.2"
   git push origin v0.2.2
   ```

4. O workflow `.github/workflows/release.yml` dispara na tag e o GoReleaser publica
   linux/darwin/windows em amd64 e arm64, mais `checksums.txt`. O changelog sai dos
   commits (sem `docs:`, `test:` e `chore:`).
5. Confira sem travar o terminal:

   ```bash
   gh run list --workflow release --limit 1
   gh release view v0.2.2 --json url,assets --jq '.url, (.assets[].name)'
   ```

6. Atualize o binário local: `go install github.com/MatheusGoncalves540/patchlore@latest`.

Regras:

- **Nunca** mova ou reenvie uma tag já publicada (force-push de tag). Se errou, publique
  a próxima versão.
- A versão do binário vem da tag: `.goreleaser.yaml` injeta `-X main.versao={{ .Version }}`.
  Quem instala com `go install ...@vX` não recebe esse ldflags, então `versaoBinario()`
  lê a versão que o Go gravou no binário (`runtime/debug`). Build local sem tag mostra a
  pseudo-versão do checkout. Não volte a chumbar versão no código.
- Os usuários recebem o prompt do tema embutido junto com o binário; quem tiver
  `~/.config/patchlore/tema.toml` continua com o override dele.

## CI de release

- Roda em `ubuntu-24.04` fixo e com versões fixas das actions/GoReleaser. A chave de
  cache do `setup-go` inclui a imagem do runner; `ubuntu-latest` mudou de versão e
  invalidou o cache, fazendo o primeiro build baixar/compilar tudo do zero (~105s).
- Cache do GitHub Actions é por ref: caches salvos numa tag só valem para aquela tag.
  Por isso o `.github/workflows/ci.yml` roda em push/PR no `main` e aquece o cache do
  branch padrão, que é o único visível de qualquer tag. Sem isso, todo release compila
  do zero (o primeiro alvo custa ~105s; os outros cinco reaproveitam o cache).
- Com cache quente o build dos 6 alvos leva dezenas de segundos; o primeiro alvo é o
  mais caro e os demais reutilizam o cache de compilação. Medido: ~3m07s sem cache
  contra ~24s com o cache do `main` restaurado.
- Para validar o pipeline sem publicar nada, rode o workflow manualmente
  (`gh workflow run release`) ou localmente:

  ```bash
  go run github.com/goreleaser/goreleaser/v2@latest check
  go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean
  ```

- `gh run watch` segura o terminal até o CI acabar. Prefira `gh run list` e
  `gh run view` quando estiver em sessão interativa.

## Armadilhas conhecidas

- Refs vindas de `.patchlore.toml` podem ser hostis: todo comando git que recebe ref usa
  `--end-of-options` (veja `internal/origem`).
- `config.SalvarRefs` edita só as linhas `antiga`/`nova` para preservar comentários e a
  seção `[tema]`; não troque por um encoder TOML.
- `patchlore ultimo` não reescreve o `.patchlore.toml`; se a ref salva sumiu, ele cai para
  `HEAD`. Se não houver execução anterior, avisa sem chamar a IA.
- `go install ...@latest` pode resolver a tag anterior por alguns minutos depois de
  publicar (o proxy cacheia a consulta). Para garantir a versão, instale `@vX.Y.Z`.
- Instalar com `go install ...@vX` recompila do código: os binários da release e o do
  `go install` são funcionalmente iguais, mas só a release leva o ldflags.
- O modo PR (`--pr`, `--comment`) e a Action nova ainda são do plano 02
  (`plans/02-go-pr-github-e-distribuicao.md`); a `action.yml` atual aponta para a versão
  Node antiga.
