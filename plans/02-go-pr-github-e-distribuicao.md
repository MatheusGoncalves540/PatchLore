# PLANO 02: PatchLore em Go — PR do GitHub, Action e distribuição
Status: pendente
Objetivo: Recuperar o modo PR (ler a PR, comentar a imagem nela) no executável Go, distribuí-lo por GitHub Releases, apontar a Action para o binário e remover a implementação Node.
Fora do escopo: mudanças no modo local e no layout (plano 01); publicação em gerenciadores de pacote (Homebrew, Scoop, winget); outros hosts além do GitHub.
Premissas e decisões: Depende do plano 01 concluído. GitHub via `net/http` puro, mesmos endpoints da versão Node (PR, commits, arquivos, refs, contents, comentários de issue). Token de `GITHUB_TOKEN`, `GH_TOKEN` ou `gh auth token`. `--pr N` (com `--repo dono/repo`, padrão do remote origin) troca a origem para a PR; sem `--pr` e sem refs, continua o modo local do plano 01. `--comment` só vale com `--pr`; a imagem vai para a branch `patchlore-assets` (configurável com `--assets-branch`) e o comentário é editado no lugar, achado por um marcador HTML oculto. GoReleaser gera binários para linux/darwin/windows em amd64 e arm64 ao publicar uma tag `v*`. A Action composite baixa o binário da release (input `version`, padrão `latest`) com `gh release download` e roda com `--pr`, sem `setup-node` nem navegador; inputs atuais (api-key, provider, model, base-url, api-type, github-token) mantidos. Node removido por inteiro (`src/`, `test/`, `package.json`, `package-lock.json`).
Pré-requisitos: Plano 01 concluído; acesso de escrita ao repositório no GitHub para criar tag e release; um repositório de teste com uma PR aberta e o secret da chave de IA.

## Tarefas
- [ ] T1 [Backend] Ler a PR via API do GitHub (token, repo do remote, achar PR pela branch) e convertê-la em `Mudancas` · arquivos: `internal/origem/github.go`, `internal/origem/github_test.go` · depende: — · verificar: `go test ./internal/origem/` (servidor `httptest` com PR de 2 páginas de commits)
- [ ] T2 [Backend] Subir o PNG na branch de assets (cria a branch se faltar) e criar ou editar o comentário marcado · arquivos: `internal/origem/comentario.go`, `internal/origem/comentario_test.go` · depende: T1 · verificar: `go test ./internal/origem/` (segunda execução edita em vez de criar)
- [ ] T3 [Backend] Adicionar `--pr`, `-r/--repo`, `-c/--comment` e `--assets-branch` ao `main.go`; `--comment` sem `--pr` vira erro claro · arquivos: `main.go` · depende: T1, T2 · verificar: `go build -o patchlore.exe . && ./patchlore.exe --pr 1 -r dono/repo-de-teste -c` comenta a imagem na PR de teste
- [ ] T4 [DevOps] Configurar GoReleaser para linux/darwin/windows × amd64/arm64 · arquivos: `.goreleaser.yaml` · depende: — · verificar: `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean` gera os 6 binários em `dist/`
- [ ] T5 [DevOps] Criar workflow de release disparado por tag `v*` · arquivos: `.github/workflows/release.yml` · depende: T4 · verificar: publicar a tag `v0.2.0-rc.1` cria a release com os 6 binários
- [ ] T6 [DevOps] Reescrever a Action para baixar o binário da release e rodar com `--pr` · arquivos: `action.yml` · depende: T3, T5 · verificar: abrir uma PR no repositório de teste usando `MatheusGoncalves540/PatchLore@<branch>` gera o comentário com a imagem em menos de 1 minuto
- [x] T7 [DevOps] Remover a implementação Node e ajustar `.gitignore` (`dist/`, `out/`, `patchlore.exe`) · arquivos: `src/cli.js`, `test/render.test.js`, `package.json`, `package-lock.json`, `.gitignore` · depende: T6 · verificar: `git ls-files "*.js" package.json` não lista nada e `go test ./...` passa (feito antes da T6 a pedido do dono; `action.yml` fica quebrado até a T6)
- [ ] T8 [Docs] Atualizar README: instalação pela release, modo PR, `--comment`, Action nova, aviso de que o `config.json` antigo precisa ser recriado com `patchlore config` · arquivos: `README.md` · depende: T7 · verificar: o exemplo de workflow do README, copiado como está num repositório de teste, comenta na PR

## Aceite final
- [ ] `go vet ./... && go test ./...` sem falhas
- [ ] `./patchlore.exe --pr <n> -r <dono/repo-de-teste> -c` duas vezes seguidas deixa um único comentário, atualizado
- [ ] A release `v0.2.0-rc.1` tem os 6 binários e a Action do README comenta na PR de teste sem instalar Node nem navegador
- [ ] `git ls-files "*.js" package.json package-lock.json` sem saída

## Riscos
- Token padrão da Action sem permissão para criar a branch de assets → README e `action.yml` documentam `contents: write` e `pull-requests: write`; erro 403 vira mensagem explicando a permissão
- Repositório do PatchLore privado bloqueia o download do binário pela Action → usar o `github-token` do input no `gh release download` e documentar a liberação de acesso
- Comentário duplicado se o marcador mudar entre versões → marcador fixo como constante única, coberto pelo teste de T2
- Remover o Node quebra quem usa `@main` → a Action só passa a apontar para o binário depois que a release existe (T6 depende de T5)

Esforço: M
