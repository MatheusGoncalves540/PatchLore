package tema

import (
	"os"
	"path/filepath"
	"testing"
)

func escrever(t *testing.T, conteudo string) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "x.toml")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func TestEmbutido(t *testing.T) {
	tm, err := Carregar("", "")
	if err != nil {
		t.Fatal(err)
	}
	if tm.LimiteItens != 6 || tm.Tags["regra"].Rotulo != "REGRA" || tm.Cores.Fundo != "#f6f6f3" {
		t.Fatalf("tema embutido inesperado: %+v", tm)
	}
	if got := tm.Nomes(); len(got) != 4 || got[0] != "correcao" {
		t.Fatalf("Nomes() = %v", got)
	}
}

func TestSobrescritaParcial(t *testing.T) {
	usuario := escrever(t, "limite_itens = 4\n[cores]\nfundo = \"#000000\"\n[tags.novo]\nfundo = \"#ff0000\"\n")
	repo := escrever(t, "antiga = \"main\"\n[tema.tags.novo]\nrotulo = \"NOVIDADE\"\n[tema.tags.extra]\nrotulo = \"EXTRA\"\n")

	tm, err := Carregar(usuario, repo)
	if err != nil {
		t.Fatal(err)
	}
	novo := tm.Tags["novo"]
	if novo.Fundo != "#ff0000" || novo.Rotulo != "NOVIDADE" {
		t.Fatalf("camadas não aplicadas: %+v", novo)
	}
	if novo.Icone == "" || novo.Cor != "#1d4ed8" {
		t.Fatalf("sobrescrita parcial perdeu campos do embutido: %+v", novo)
	}
	if tm.Cores.Fundo != "#000000" || tm.Cores.Texto != "#16181d" || tm.LimiteItens != 4 {
		t.Fatalf("cores/limite: %+v %d", tm.Cores, tm.LimiteItens)
	}
	if tm.Tags["extra"].Rotulo != "EXTRA" || tm.Tags["regra"].Rotulo != "REGRA" {
		t.Fatalf("tags: %+v", tm.Tags)
	}
}

func TestArquivoInvalido(t *testing.T) {
	if _, err := Carregar(escrever(t, "limite_itens = \"x\""), ""); err == nil {
		t.Fatal("esperava erro com TOML inválido")
	}
}
