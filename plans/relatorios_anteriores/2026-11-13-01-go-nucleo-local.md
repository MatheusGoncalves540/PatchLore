# Relatório: plano 01 (PatchLore em Go, núcleo e modo local)
Branch `plano/01-go-nucleo-local` · status: parcial (14/14 tarefas; 2 itens de aceite dependem do dono)

## Concluído
- Base: go.mod, main.go, referência PNG gerada pela versão Node (`testdata/referencia.png`).
- Domínio: notas + Validar, tema em camadas, config de provedores (600) e `.patchlore.toml`.
- Origem local via binário git; IA Anthropic (saída estruturada + fallback na API oficial) e OpenAI-compatível; modelos ao vivo.
- Render canvas → PNG 2x (1920 px) + PDF; Markdown e notes.json; menus huh; CLI completa; teste ponta a ponta; README.

## Validação
- `go vet ./... && go test ./...`: tudo passa. Binário: 20,2 MB.
- README executado num repo descartável: refs explícitas, reuso sem argumentos, tema do repo muda PNG/PDF, tag nova do exemplo renderiza, zero commits sai com código 1.

## Pendente com o dono
- Aceite 3: rodar com IA real (`patchlore config` ou `PATCHLORE_API_KEY`). Não usei a chave do ambiente para não gastar sem pedido.
- Aceite 4: aprovação visual. A referência saiu em Segoe UI (o Chrome não tinha Inter), então as quebras de linha diferem um pouco; hierarquia, cores e tags batem.
- T11: o menu abre (conferido); salvar pelo menu não dá para automatizar aqui (huh pede console real). A gravação em si é coberta por `config_test`.

## Premissas que mudaram o resultado
- Inter Bold embutida além de Regular/SemiBold: h1 e h2 do layout original são bold (700). +420 KB.
- Pílula e rodapé usam `antiga → nova`; o rodapé mostra `-t` quando informado.
- `.patchlore.toml` é salvo também quando as refs vêm por argumento (o aceite 3 exige o reuso).
- Letter-spacing do rótulo do topo não existe no canvas; ficou sem.

## Ressalvas
- Segurança: a seção `[tema]` do repo pode trocar o prompt. Ok no modo local; no modo PR (plano 02) quem abre a PR pode editar `.patchlore.toml` → ler o tema da branch base.
- `.gitignore` ainda não ignora `patchlore.exe`/`out/` (é a T7 do plano 02). Há uma alteração sua não commitada no `.gitignore` (`.claude/`), que deixei intacta.
