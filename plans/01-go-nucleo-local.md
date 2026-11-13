# PLANO 01: PatchLore em Go — núcleo e modo local
Status: pendente
Objetivo: Ter o executável Go `patchlore` gerando notes.json, PNG, PDF e Markdown a partir de duas refs de um repositório git local, sem Node nem navegador.
Fora do escopo: origem PR do GitHub, comentário na PR, GitHub Action, GoReleaser e remoção do código Node (todos no plano 02); SVG; migração automática do antigo `config.json`.
Premissas e decisões: Go 1.26, módulo `github.com/MatheusGoncalves540/patchlore`, `main.go` na raiz e pacotes em `internal/` com nomes em PT-BR (notas, tema, config, origem, ia, render, menu). Dependências permitidas: `tdewolff/canvas` (desenho + PNG + PDF a partir do mesmo layout; importar só o núcleo, o rasterizador e o renderer PDF), `BurntSushi/toml`, `charmbracelet/huh`; o resto é biblioteca padrão (`net/http`, `encoding/json`, `flag`, `os/exec`, `embed`). Sem SDK de IA. Fonte Inter (licença OFL) embutida com `go:embed`. Código legível, funções curtas, comentários em PT-BR explicando o porquê com voz humana (nada de comentário que repete o código). Tema (cores, tags com rótulo/cor/ícone, limite de itens, prompt do sistema) em `internal/tema/tema.toml` embutido; o usuário sobrescreve com `~/.config/patchlore/tema.toml` e o repo com a seção `[tema]` do `.patchlore.toml` (repo vence usuário, usuário vence embutido). Provedores e chaves ficam só em `~/.config/patchlore/config.toml` (permissão 600), nunca no `.patchlore.toml`. Variáveis `PATCHLORE_API_KEY`, `PATCHLORE_PROVIDER`, `PATCHLORE_MODEL`, `PATCHLORE_BASE_URL`, `PATCHLORE_API_TYPE` vencem a config salva. Presets: claude (anthropic, `https://api.anthropic.com`, `claude-opus-5-5`) e deepseek (openai, `https://api.deepseek.com`, `deepseek-chat`). Origem local: `patchlore antiga..nova`; commits de `git log antiga..nova --no-merges`; arquivos de `git diff --numstat antiga...nova`; chama o binário `git`, sem `git fetch`; sem argumentos e em terminal, menu huh lista branches locais, remotas e tags; a escolha é salva em `.patchlore.toml` na raiz do repo e reutilizada; zero commits entre as refs → aviso e saída com código 1. Título/descrição no modo local: flag opcional `-t`; o rodapé usa `antiga → nova`. Validação das notas com struct + `Validar()` contra o tema (tags válidas, 1..N itens, campos não vazios); o JSON Schema enviado à IA é montado a partir do tema. Layout replica o visual atual (largura 960 pt, escala 2x no PNG = 1920 px); a altura é calculada a partir do conteúdo. Sem migração do `config.json` antigo: `patchlore config` recria (versão 0.x).
Pré-requisitos: Go 1.26+ e git no PATH; uma chave de IA para o teste manual com IA real; referência de UI = PNG gerado pela versão Node atual (`patchlore-out/patchnotes.png`), guardado em `testdata/referencia.png` antes de começar.

## Tarefas
### Fase 1: base
- [x] T1 [Arquiteto] Criar `go.mod`, `main.go` com ajuda (`-h`) e comentário de pacote explicando o fluxo; adicionar `testdata/referencia.png` · arquivos: `go.mod`, `main.go`, `testdata/referencia.png` · depende: — · verificar: `go build -o patchlore.exe . && ./patchlore.exe -h` imprime o uso
- [x] T2 [Backend] Criar tipos `Mudancas` e `Notas`, `Validar(notas, tema)` e exemplo de notas · arquivos: `internal/notas/notas.go`, `internal/notas/notas_test.go`, `testdata/notes.json` · depende: T1 · verificar: `go test ./internal/notas/` (cobre tag inválida, 0 itens, itens acima do limite, campo vazio)
- [x] T3 [Backend] Criar `tema.toml` embutido e carregamento em camadas (embutido → usuário → repo) · arquivos: `internal/tema/tema.go`, `internal/tema/tema.toml`, `internal/tema/tema_test.go` · depende: T1 · verificar: `go test ./internal/tema/` (sobrescrita parcial mantém o resto do embutido)
- [x] T4 [Backend] Ler e salvar provedores em `config.toml` (600), presets e sobrescrita por variáveis de ambiente · arquivos: `internal/config/config.go`, `internal/config/config_test.go` · depende: T1 · verificar: `go test ./internal/config/`
### Fase 2: origem local
- [ ] T5 [Backend] Ler o repositório local via binário git: listar refs, commits `antiga..nova --no-merges`, numstat · arquivos: `internal/origem/git.go`, `internal/origem/git_test.go` · depende: T2 · verificar: `go test ./internal/origem/` (o teste cria um repositório temporário com 2 branches e 1 merge)
- [ ] T6 [Backend] Ler e salvar `.patchlore.toml` do projeto (refs + seção `[tema]`) · arquivos: `internal/config/projeto.go`, `internal/config/projeto_test.go` · depende: T4 · verificar: `go test ./internal/config/`
### Fase 3: IA
- [ ] T7 [Backend] Montar prompt e JSON Schema a partir do tema; chamar a API da Anthropic via `net/http` com saída estruturada · arquivos: `internal/ia/ia.go`, `internal/ia/anthropic.go`, `internal/ia/ia_test.go` · depende: T2, T3 · verificar: `go test ./internal/ia/` (servidor `httptest` devolve resposta válida, recusa e JSON fora do formato)
- [ ] T8 [Backend] Chamar APIs compatíveis com OpenAI (`/chat/completions`, aceita cerca ```json) e listar modelos ao vivo nos dois formatos · arquivos: `internal/ia/openai.go`, `internal/ia/modelos.go`, `internal/ia/openai_test.go` · depende: T7 · verificar: `go test ./internal/ia/`
### Fase 4: saída
- [ ] T9 [Backend] Embutir a Inter e desenhar o layout com canvas, exportando PNG (2x) e PDF do mesmo desenho · arquivos: `internal/render/render.go`, `internal/render/fontes/Inter-Regular.ttf`, `internal/render/fontes/Inter-SemiBold.ttf`, `internal/render/render_test.go` · depende: T2, T3 · verificar: `go test ./internal/render/` (PNG com 1920 px de largura, PDF começa com `%PDF`, texto de 240 caracteres quebra sem estourar o card)
- [ ] T10 [Backend] Gerar Markdown e `notes.json` · arquivos: `internal/render/markdown.go`, `internal/render/markdown_test.go` · depende: T2, T3 · verificar: `go test ./internal/render/`
### Fase 5: CLI
- [ ] T11 [Backend] Criar menus huh: escolher refs (antiga e nova) e gerenciar provedores (adicionar, editar, remover, padrão, modelos ao vivo com fallback para digitação) · arquivos: `internal/menu/refs.go`, `internal/menu/provedores.go` · depende: T4, T5, T8 · verificar: `go build -o patchlore.exe . && ./patchlore.exe config` abre o menu e salva `~/.config/patchlore/config.toml`
- [ ] T12 [Backend] Ligar tudo no `main.go`: `antiga..nova` e flags `-n -o -p -m -t --open`; menu sem argumentos em TTY, erro claro sem TTY · arquivos: `main.go` · depende: T6, T9, T10, T11 · verificar: `go build -o patchlore.exe . && ./patchlore.exe -n testdata/notes.json HEAD~3..HEAD -o out` cria `out/notes.json`, `out/patchnotes.png`, `out/patchnotes.pdf` e `out/patchnotes.md`
- [ ] T13 [QA] Teste ponta a ponta: repo git temporário + IA falsa (`httptest`) passando pelo `main` · arquivos: `main_test.go` · depende: T12 · verificar: `go test -run TestPontaAPonta .`
- [ ] T14 [Docs] Atualizar README: instalação Go, uso local com refs, `.patchlore.toml`, como editar o tema, provedores e variáveis de ambiente · arquivos: `README.md` · depende: T12 · verificar: cada comando do README roda como descrito num repo de teste

## Aceite final
- [ ] `go vet ./... && go test ./...` sem falhas
- [ ] `go build -o patchlore.exe .` gera executável com menos de 25 MB
- [ ] Num repo com duas branches, `./patchlore.exe production..development` com IA real gera os 4 arquivos e a segunda execução sem argumentos reaproveita as refs do `.patchlore.toml`
- [ ] PNG gerado comparado lado a lado com `testdata/referencia.png`: mesma hierarquia, cores e tags (aprovação visual do dono)
- [ ] Mudar a cor de uma tag em `.patchlore.toml` muda o PNG e o PDF sem recompilar (`./patchlore.exe -n testdata/notes.json HEAD~3..HEAD`)

## Riscos
- Layout do canvas diverge do HTML atual (quebra de linha, espaçamento) → T9 compara com `testdata/referencia.png` e ajusta constantes de espaçamento no tema
- `tdewolff/canvas` puxa dependências pesadas → importar só núcleo, rasterizador e PDF; o aceite trava em 25 MB
- API da Anthropic rejeita palavras-chave do JSON Schema (minItems/maxItems) → schema enxuto na API e limites garantidos por `Validar()`
- `git` ausente no PATH ou pasta fora de repo → mensagem clara antes de qualquer chamada à IA
- huh com terminal sem suporte (CI, pipe) → detectar TTY e exigir refs por argumento

Esforço: G
