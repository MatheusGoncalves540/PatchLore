// O patchlore transforma o histórico entre duas refs de um repositório git em
// patchnotes visuais. O fluxo é sempre o mesmo: ler as mudanças (commits e
// arquivos), pedir à IA só o texto num JSON de formato fixo, validar esse JSON
// contra o tema e desenhar o resultado em PNG, PDF e Markdown. A IA nunca toca
// no visual; é isso que garante que todo patchnotes saia com a mesma cara.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
	"github.com/MatheusGoncalves540/patchlore/internal/estado"
	"github.com/MatheusGoncalves540/patchlore/internal/ia"
	"github.com/MatheusGoncalves540/patchlore/internal/menu"
	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/origem"
	"github.com/MatheusGoncalves540/patchlore/internal/render"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

const uso = `patchlore: patchnotes visuais a partir do histórico do git

Uso
  patchlore antiga..nova    gera o patchnotes do que entrou em nova desde antiga
  patchlore ultimo          gera do último commit processado neste repo até agora
  patchlore                 reaproveita as refs do .patchlore.toml (ou abre o menu)
  patchlore config          configura provedores de IA (Claude, DeepSeek, outros)
  patchlore mcp             servidor MCP (stdio) para editores como Zed e Claude

Opções
`

// versao é injetada pelo GoReleaser na publicação (-X main.versao). Quem
// instala com `go install ...@vX` compila do código e não recebe esse valor;
// nesse caso a versão que o Go gravou no binário é lida de runtime/debug.
// Build sem tag mostra a pseudo-versão do checkout (ou "dev" se não houver).
var versao = "dev"

func versaoBinario() string {
	v := versao
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			v = info.Main.Version
		}
	}
	if v = strings.TrimPrefix(v, "v"); v == "" || v == "(devel)" {
		return "dev"
	}
	return v
}

type opcoes struct {
	notas, saida, provedor, modelo, titulo string
	abrir                                  bool
	notasProntas                           []byte // notas já em memória (vindas do MCP)
}

func main() {
	err := rodar(os.Args[1:], os.Stdout)
	switch {
	case errors.Is(err, menu.Cancelado):
		fmt.Fprintln(os.Stderr, "Cancelado.")
		os.Exit(1)
	case err != nil:
		fmt.Fprintln(os.Stderr, "patchlore:", err)
		os.Exit(1)
	}
}

func rodar(args []string, saida io.Writer) error {
	var o opcoes
	fs := flag.NewFlagSet("patchlore", flag.ContinueOnError)
	fs.StringVar(&o.notas, "n", "", "usa um notes.json pronto em vez de chamar a IA")
	fs.StringVar(&o.saida, "o", "patchlore-out", "pasta de saída")
	fs.StringVar(&o.provedor, "p", "", "provedor salvo (padrão: o marcado com ★)")
	fs.StringVar(&o.modelo, "m", "", "troca o modelo só nesta execução")
	fs.StringVar(&o.titulo, "t", "", "título do rodapé e do prompt (padrão: antiga → nova)")
	fs.BoolVar(&o.abrir, "open", false, "abre o PNG ao terminar")
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), uso)
		fs.PrintDefaults()
	}
	posicionais, err := lerFlags(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}

	switch {
	case len(posicionais) > 1:
		return fmt.Errorf("argumentos demais: %v (use antiga..nova)", posicionais)
	case len(posicionais) == 1 && posicionais[0] == "config":
		return menu.Provedores(config.Caminho())
	case len(posicionais) == 1 && posicionais[0] == "mcp":
		return servirMCP(os.Stdin, os.Stdout)
	}
	// Em `patchlore ultimo` a ref vem do estado salvo, não da linha de comando.
	continuar := len(posicionais) == 1 && posicionais[0] == "ultimo"
	ref := ""
	if len(posicionais) == 1 && !continuar {
		ref = posicionais[0]
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return gerar(context.Background(), dir, ref, continuar, o, saida)
}

// O pacote flag para no primeiro argumento solto; aqui as flags podem vir
// antes ou depois das refs (patchlore main..dev -o saida).
func lerFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var posicionais []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return posicionais, nil
		}
		posicionais = append(posicionais, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func gerar(ctx context.Context, dir, ref string, continuar bool, o opcoes, saida io.Writer) error {
	// Repo e git primeiro: se faltar algo aqui, nem chegamos a gastar com a IA.
	raiz, err := origem.Raiz(dir)
	if err != nil {
		return err
	}
	arquivoProjeto := config.CaminhoProjeto(raiz)
	tm, err := tema.Carregar(filepath.Join(config.Dir(), "tema.toml"), arquivoProjeto)
	if err != nil {
		return err
	}

	antiga, nova, err := resolverRefs(raiz, arquivoProjeto, ref, continuar, saida)
	if err != nil {
		return err
	}
	m, err := origem.Local(raiz, antiga, nova)
	if errors.Is(err, origem.ErrSemCommits) {
		return fmt.Errorf("aviso: %w; nada para resumir", err)
	}
	if err != nil {
		return err
	}
	// O estado substitui as refs salvas em `ultimo`; reescrever o
	// .patchlore.toml aí trocaria as refs nomeadas por um SHA.
	if !continuar {
		if err := config.SalvarRefs(arquivoProjeto, antiga, nova); err != nil {
			return err
		}
	}
	m.Titulo = o.titulo
	fmt.Fprintf(saida, "%s: %d commits · %d arquivos\n", render.Refs(m), len(m.Commits), len(m.Arquivos))

	n, err := obterNotas(ctx, o, m, tm, saida)
	if err != nil {
		return err
	}
	if err := salvar(n, m, tm, o, saida); err != nil {
		return err
	}
	lembrar(raiz, nova, saida)
	return nil
}

// lembrar guarda o commit final para o próximo `patchlore ultimo`. Se der
// errado, o patchnotes já está salvo: avisamos e seguimos.
func lembrar(raiz, ref string, saida io.Writer) {
	commit, err := origem.Resolve(raiz, ref)
	if err != nil {
		fmt.Fprintf(saida, "aviso: não consegui guardar até onde paramos: %v\n", err)
		return
	}
	if err := estado.Salvar(raiz, estado.Estado{Ref: ref, Commit: commit}); err != nil {
		fmt.Fprintf(saida, "aviso: não consegui guardar até onde paramos: %v\n", err)
	}
}

// resolverRefs: argumento explícito vence; com continuar, vale o commit da
// última execução salva no estado; sem argumento, o que o projeto guardou da
// última vez; sem nada guardado, pergunta (só se houver terminal).
func resolverRefs(raiz, arquivoProjeto, ref string, continuar bool, saida io.Writer) (string, string, error) {
	if continuar {
		e, err := estado.Ler(raiz)
		if err != nil {
			return "", "", err
		}
		if e.Commit == "" {
			return "", "", errors.New("nenhuma execução anterior neste repositório; rode `patchlore antiga..nova` primeiro")
		}
		// A ref pode ter sido apagada (merge fechou a branch); nesse caso o
		// que está em HEAD é o que a pessoa quer resumir agora.
		nova := e.Ref
		if nova == "" {
			nova = "HEAD"
		} else if _, err := origem.Resolve(raiz, nova); err != nil {
			nova = "HEAD"
		}
		fmt.Fprintf(saida, "Continuando de %s até %s\n", curto(e.Commit), nova)
		return e.Commit, nova, nil
	}
	if ref != "" {
		antiga, nova, ok := strings.Cut(ref, "..")
		if !ok || antiga == "" || nova == "" || strings.HasPrefix(nova, ".") {
			return "", "", fmt.Errorf("refs inválidas %q: use antiga..nova (ex.: production..development)", ref)
		}
		return antiga, nova, nil
	}
	salvo, err := config.LerProjeto(arquivoProjeto)
	if err != nil {
		return "", "", err
	}
	if salvo.Antiga != "" && salvo.Nova != "" {
		fmt.Fprintf(saida, "Usando %s..%s do %s\n", salvo.Antiga, salvo.Nova, config.ArquivoProjeto)
		return salvo.Antiga, salvo.Nova, nil
	}
	if !terminal() {
		return "", "", errors.New("sem terminal para o menu; passe as refs: patchlore antiga..nova")
	}
	refs, err := origem.Refs(raiz)
	if err != nil {
		return "", "", err
	}
	return menu.EscolherRefs(refs, "", "")
}

// curto deixa o SHA legível no terminal e no título do patchnotes.
func curto(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// terminal diz se dá para abrir menu: precisa de entrada e saída interativas.
func terminal() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		info, err := f.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

func obterNotas(ctx context.Context, o opcoes, m notas.Mudancas, tm tema.Tema, saida io.Writer) (notas.Notas, error) {
	if o.notasProntas != nil {
		return notas.Decodificar(o.notasProntas, tm)
	}
	if o.notas != "" {
		dados, err := os.ReadFile(o.notas)
		if err != nil {
			return notas.Notas{}, err
		}
		fmt.Fprintf(saida, "Usando %s\n", o.notas)
		return notas.Decodificar(dados, tm)
	}
	cfg, err := config.Ler(config.Caminho())
	if err != nil {
		return notas.Notas{}, err
	}
	p, err := cfg.Resolver(o.provedor, o.modelo, os.Getenv)
	if err != nil {
		return notas.Notas{}, err
	}
	fmt.Fprintf(saida, "Resumindo com %s · %s…\n", p.Nome, p.Modelo)
	return ia.Resumir(ctx, p, m, tm)
}

func salvar(n notas.Notas, m notas.Mudancas, tm tema.Tema, o opcoes, saida io.Writer) error {
	pngBytes, pdfBytes, err := render.Imagens(n, m, tm)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(o.saida, 0o755); err != nil {
		return err
	}
	arquivos := map[string][]byte{
		"notes.json":     render.NotasJSON(n),
		"patchnotes.png": pngBytes,
		"patchnotes.pdf": pdfBytes,
		"patchnotes.md":  []byte(render.Markdown(n, m, tm)),
	}
	for nome, dados := range arquivos {
		if err := os.WriteFile(filepath.Join(o.saida, nome), dados, 0o644); err != nil {
			return err
		}
	}

	fmt.Fprintf(saida, "\n%s\n", n.Titulo)
	for _, it := range n.Itens {
		fmt.Fprintf(saida, "  %-9s %s\n", tm.Tags[it.Tag].Rotulo, it.Titulo)
	}
	absoluto, _ := filepath.Abs(o.saida)
	fmt.Fprintf(saida, "\n✔ %s  (edite notes.json e rode com -n para ajustar o texto)\n", absoluto)
	if o.abrir {
		abrir(filepath.Join(absoluto, "patchnotes.png"))
	}
	return nil
}

func abrir(caminho string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", caminho)
	case "darwin":
		cmd = exec.Command("open", caminho)
	default:
		cmd = exec.Command("xdg-open", caminho)
	}
	_ = cmd.Start() // abrir é cortesia; se falhar, os arquivos já estão salvos
}
