package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func semEnv(string) string { return "" }

func cfgExemplo() Config {
	ds := Presets["deepseek"]
	ds.Chave = "k1"
	return Config{Padrao: "deepseek", Provedores: map[string]Provedor{
		"deepseek": ds,
		"meu":      {Nome: "meu", Tipo: TipoOpenAI, URL: "http://x", Modelo: "m", Chave: "k2"},
	}}
}

func TestSalvarELer(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := Salvar(caminho, cfgExemplo()); err != nil {
		t.Fatal(err)
	}
	c, err := Ler(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if c.Padrao != "deepseek" || c.Provedores["meu"].Chave != "k2" || c.Provedores["meu"].Nome != "meu" {
		t.Fatalf("ida e volta: %+v", c)
	}
	if runtime.GOOS != "windows" { // o Windows não tem bits de permissão Unix
		if info, _ := os.Stat(caminho); info.Mode().Perm() != 0o600 {
			t.Fatalf("permissão %v, esperava 600", info.Mode().Perm())
		}
	}
}

func TestLerSemArquivo(t *testing.T) {
	c, err := Ler(filepath.Join(t.TempDir(), "nada.toml"))
	if err != nil || c.Provedores == nil {
		t.Fatalf("primeira execução deveria dar config vazia: %v", err)
	}
}

func TestResolver(t *testing.T) {
	c := cfgExemplo()
	if p, _ := c.Resolver("", "", semEnv); p.Nome != "deepseek" {
		t.Errorf("padrão: %+v", p)
	}
	if p, _ := c.Resolver("meu", "", semEnv); p.Chave != "k2" {
		t.Errorf("--provider: %+v", p)
	}
	if p, _ := c.Resolver("", "deepseek-reasoner", semEnv); p.Modelo != "deepseek-reasoner" {
		t.Errorf("--model: %+v", p)
	}
	if c.Provedores["deepseek"].Modelo != "deepseek-chat" {
		t.Error("--model não pode alterar a config salva")
	}
}

func TestResolverEnvVence(t *testing.T) {
	env := map[string]string{"PATCHLORE_API_KEY": "env", "PATCHLORE_PROVIDER": "claude", "PATCHLORE_MODEL": "outro"}
	p, err := cfgExemplo().Resolver("meu", "", func(k string) string { return env[k] })
	if err != nil || p.Tipo != TipoAnthropic || p.URL != "https://api.anthropic.com" || p.Chave != "env" || p.Modelo != "outro" {
		t.Fatalf("env: %+v %v", p, err)
	}
}

func TestResolverErros(t *testing.T) {
	if _, err := (Config{}).Resolver("", "", semEnv); err == nil || !strings.Contains(err.Error(), "patchlore config") {
		t.Errorf("sem provedor: %v", err)
	}
	if _, err := cfgExemplo().Resolver("nada", "", semEnv); err == nil || !strings.Contains(err.Error(), "não existe") {
		t.Errorf("provedor inexistente: %v", err)
	}
}
