package menu

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
	"github.com/MatheusGoncalves540/patchlore/internal/ia"
)

const outroModelo = "\x00outro" // valor que nenhum modelo real usa

// Provedores é o `patchlore config`: adicionar, editar, remover e escolher o
// padrão. Cada mudança é salva na hora, então sair no meio não perde nada.
func Provedores(caminho string) error {
	cfg, err := config.Ler(caminho)
	if err != nil {
		return err
	}
	for {
		fmt.Println("\n" + resumo(cfg))
		acoes := []huh.Option[string]{
			huh.NewOption("Adicionar Claude (api.anthropic.com)", "claude"),
			huh.NewOption("Adicionar DeepSeek (api.deepseek.com)", "deepseek"),
			huh.NewOption("Adicionar outro (qualquer URL)", "outro"),
		}
		if len(cfg.Provedores) > 0 {
			acoes = append(acoes,
				huh.NewOption("Editar (URL, modelo, nome, chave)", "editar"),
				huh.NewOption("Definir padrão", "padrao"),
				huh.NewOption("Remover", "remover"))
		}
		acoes = append(acoes, huh.NewOption("Sair", "sair"))

		var acao string
		if err := huh.NewSelect[string]().Title("O que fazer?").Options(acoes...).Value(&acao).Run(); err != nil {
			return err
		}
		switch acao {
		case "sair":
			fmt.Println("Salvo em " + caminho)
			return nil
		case "claude", "deepseek":
			base := config.Presets[acao]
			if _, existe := cfg.Provedores[base.Nome]; existe {
				base.Nome += "-2"
			}
			err = editar(&cfg, base, false)
		case "outro":
			err = editar(&cfg, config.Provedor{}, false)
		case "editar":
			var nome string
			if nome, err = escolher(cfg, "Qual editar?"); err == nil {
				err = editar(&cfg, cfg.Provedores[nome], true)
			}
		case "padrao":
			if cfg.Padrao, err = escolher(cfg, "Qual será o padrão?"); err == nil {
				err = config.Salvar(caminho, cfg)
			}
		case "remover":
			err = remover(&cfg)
		}
		if err == nil && acao != "padrao" {
			err = config.Salvar(caminho, cfg)
		}
		if err != nil {
			return err
		}
	}
}

// A chave aparece só com os 4 últimos caracteres: dá para reconhecer qual é
// sem expor o segredo na tela (ou num print).
func resumo(cfg config.Config) string {
	if len(cfg.Provedores) == 0 {
		return "Nenhum provedor ainda."
	}
	nomes := nomesOrdenados(cfg)
	var b strings.Builder
	b.WriteString("Provedores:\n")
	for _, n := range nomes {
		p := cfg.Provedores[n]
		marca := " "
		if n == cfg.Padrao {
			marca = "★"
		}
		chave := "sem chave"
		if len(p.Chave) >= 4 {
			chave = "••••" + p.Chave[len(p.Chave)-4:]
		}
		fmt.Fprintf(&b, "%s %-12s %-22s %s  %s\n", marca, n, p.Modelo, p.URL, chave)
	}
	return strings.TrimRight(b.String(), "\n")
}

func nomesOrdenados(cfg config.Config) []string {
	nomes := make([]string, 0, len(cfg.Provedores))
	for n := range cfg.Provedores {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	return nomes
}

func escolher(cfg config.Config, titulo string) (string, error) {
	var nome string
	err := huh.NewSelect[string]().Title(titulo).Options(huh.NewOptions(nomesOrdenados(cfg)...)...).Value(&nome).Run()
	return nome, err
}

var nomeValido = regexp.MustCompile(`^[\w.-]+$`)

func editar(cfg *config.Config, base config.Provedor, editando bool) error {
	p := base
	err := huh.NewInput().Title("Nome").Value(&p.Nome).Validate(func(v string) error {
		if !nomeValido.MatchString(v) {
			return errors.New("use letras, números, - _ .")
		}
		if _, existe := cfg.Provedores[v]; existe && v != base.Nome {
			return errors.New("já existe um provedor com esse nome")
		}
		return nil
	}).Run()
	if err != nil {
		return err
	}
	if p.Tipo == "" {
		err = huh.NewSelect[string]().Title("Formato da API").Options(
			huh.NewOption("Compatível com OpenAI (/chat/completions: DeepSeek, OpenRouter, Groq, Ollama…)", config.TipoOpenAI),
			huh.NewOption("Anthropic (/v1/messages)", config.TipoAnthropic),
		).Value(&p.Tipo).Run()
		if err != nil {
			return err
		}
	}
	err = huh.NewInput().Title("URL base da API").Value(&p.URL).Validate(func(v string) error {
		if u, err := url.Parse(v); err != nil || u.Scheme == "" || u.Host == "" {
			return errors.New("URL inválida")
		}
		return nil
	}).Run()
	if err != nil {
		return err
	}
	p.URL = strings.TrimRight(p.URL, "/")

	titulo := "Chave da API"
	if editando && base.Chave != "" {
		titulo = "Chave da API (Enter mantém a atual)"
	}
	var chave string
	err = huh.NewInput().Title(titulo).EchoMode(huh.EchoModePassword).Value(&chave).Validate(func(v string) error {
		if v == "" && !(editando && base.Chave != "") {
			return errors.New("obrigatória")
		}
		return nil
	}).Run()
	if err != nil {
		return err
	}
	if chave != "" {
		p.Chave = chave
	}
	if p.Modelo, err = escolherModelo(p); err != nil {
		return err
	}

	if editando && p.Nome != base.Nome {
		delete(cfg.Provedores, base.Nome)
		if cfg.Padrao == base.Nome {
			cfg.Padrao = p.Nome
		}
	}
	cfg.Provedores[p.Nome] = p
	if cfg.Padrao == "" {
		cfg.Padrao = p.Nome
	}
	fmt.Printf("%s salvo\n", p.Nome)
	return nil
}

// escolherModelo busca a lista ao vivo; se a API não responder (rede, chave
// sem permissão de listar), a pessoa digita o nome e segue a vida.
func escolherModelo(p config.Provedor) (string, error) {
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	fmt.Printf("Buscando modelos em %s…\n", p.URL)
	modelos, err := ia.Modelos(ctx, p)
	if err != nil {
		fmt.Printf("Não deu para listar os modelos (%.80s). Digite o nome.\n", err.Error())
	}
	modelo := p.Modelo
	if len(modelos) > 0 {
		opcoes := huh.NewOptions(modelos...)
		opcoes = append(opcoes, huh.NewOption("Outro…", outroModelo))
		if err := huh.NewSelect[string]().Title("Modelo (/ para filtrar)").Options(opcoes...).
			Filtering(true).Height(12).Value(&modelo).Run(); err != nil {
			return "", err
		}
		if modelo != outroModelo {
			return modelo, nil
		}
		modelo = p.Modelo
	}
	err = huh.NewInput().Title("Modelo").Value(&modelo).Validate(func(v string) error {
		if strings.TrimSpace(v) == "" {
			return errors.New("obrigatório")
		}
		return nil
	}).Run()
	return strings.TrimSpace(modelo), err
}

func remover(cfg *config.Config) error {
	nome, err := escolher(*cfg, "Qual remover?")
	if err != nil {
		return err
	}
	var ok bool
	if err := huh.NewConfirm().Title("Remover " + nome + "?").Value(&ok).Run(); err != nil || !ok {
		return err
	}
	delete(cfg.Provedores, nome)
	if cfg.Padrao == nome {
		cfg.Padrao = ""
		if nomes := nomesOrdenados(*cfg); len(nomes) > 0 {
			cfg.Padrao = nomes[0]
		}
	}
	return nil
}
