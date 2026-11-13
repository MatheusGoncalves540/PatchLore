// Pacote config cuida de duas coisas que não podem se misturar: a config do
// usuário (provedores e chaves, em ~/.config/patchlore) e a do projeto
// (.patchlore.toml, que vai para o git e por isso nunca leva chave).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Tipo diz qual formato de API o provedor fala.
const (
	TipoAnthropic = "anthropic"
	TipoOpenAI    = "openai" // qualquer API compatível com /chat/completions
)

type Provedor struct {
	Nome   string `toml:"-"`
	Tipo   string `toml:"tipo"`
	URL    string `toml:"url"`
	Chave  string `toml:"chave"`
	Modelo string `toml:"modelo"`
}

type Config struct {
	Padrao     string              `toml:"padrao"`
	Provedores map[string]Provedor `toml:"provedores"`
}

var Presets = map[string]Provedor{
	"claude":   {Nome: "claude", Tipo: TipoAnthropic, URL: "https://api.anthropic.com", Modelo: "claude-opus-5-5"},
	"deepseek": {Nome: "deepseek", Tipo: TipoOpenAI, URL: "https://api.deepseek.com", Modelo: "deepseek-chat"},
}

// Dir segue o XDG mesmo no Windows: um único lugar fácil de achar e de
// documentar vale mais que seguir a convenção de cada sistema.
func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "patchlore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "patchlore")
}

func Caminho() string { return filepath.Join(Dir(), "config.toml") }

// Ler devolve config vazia se o arquivo não existe: primeira execução não é erro.
func Ler(caminho string) (Config, error) {
	c := Config{Provedores: map[string]Provedor{}}
	_, err := toml.DecodeFile(caminho, &c)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("config inválida em %s: %w", caminho, err)
	}
	if c.Provedores == nil {
		c.Provedores = map[string]Provedor{}
	}
	for nome, p := range c.Provedores {
		p.Nome = nome
		c.Provedores[nome] = p
	}
	return c, nil
}

// Salvar grava com permissão 600 porque o arquivo guarda chaves de API.
func Salvar(caminho string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	if err := toml.NewEncoder(&b).Encode(c); err != nil {
		return err
	}
	if err := os.WriteFile(caminho, []byte(b.String()), 0o600); err != nil {
		return err
	}
	// WriteFile não mexe na permissão de um arquivo que já existia.
	return os.Chmod(caminho, 0o600)
}

// Resolver escolhe o provedor desta execução. Variáveis de ambiente vencem
// tudo (é assim que CI e a Action configuram); senão vale o nome pedido ou o
// padrão salvo. modelo, se vier, troca o modelo só nesta execução.
func (c Config) Resolver(nome, modelo string, env func(string) string) (Provedor, error) {
	var p Provedor
	if chave := env("PATCHLORE_API_KEY"); chave != "" {
		preset := env("PATCHLORE_PROVIDER")
		if preset == "" {
			preset = "claude"
		}
		p = Presets[preset]
		p.Nome = preset
		p.Chave = chave
		sobrescrever(&p.Tipo, env("PATCHLORE_API_TYPE"))
		sobrescrever(&p.URL, env("PATCHLORE_BASE_URL"))
		sobrescrever(&p.Modelo, env("PATCHLORE_MODEL"))
		if p.Tipo == "" {
			p.Tipo = TipoOpenAI
		}
	} else {
		if nome == "" {
			nome = c.Padrao
		}
		if nome == "" {
			return p, errors.New("nenhum provedor configurado; rode `patchlore config`")
		}
		var ok bool
		if p, ok = c.Provedores[nome]; !ok {
			return p, fmt.Errorf("provedor %q não existe; rode `patchlore config`", nome)
		}
	}
	sobrescrever(&p.Modelo, modelo)
	if p.URL == "" || p.Modelo == "" {
		return p, fmt.Errorf("provedor %q sem URL ou modelo", p.Nome)
	}
	return p, nil
}

func sobrescrever(campo *string, valor string) {
	if valor != "" {
		*campo = valor
	}
}
