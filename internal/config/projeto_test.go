package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjetoIdaEVolta(t *testing.T) {
	caminho := CaminhoProjeto(t.TempDir())
	if p, err := LerProjeto(caminho); err != nil || p.Antiga != "" {
		t.Fatalf("sem arquivo deveria vir vazio: %+v %v", p, err)
	}
	if err := SalvarRefs(caminho, "production", "development"); err != nil {
		t.Fatal(err)
	}
	if p, _ := LerProjeto(caminho); p.Antiga != "production" || p.Nova != "development" {
		t.Fatalf("lido: %+v", p)
	}
}

func TestSalvarRefsPreservaTema(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), ArquivoProjeto)
	original := "# minhas refs\nantiga = \"a\"\nnova = \"b\"\n\n[tema.tags.novo]\n# azul da marca\nfundo = \"#0000ff\"\n"
	if err := os.WriteFile(caminho, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SalvarRefs(caminho, "v1", "v2"); err != nil {
		t.Fatal(err)
	}
	dados, _ := os.ReadFile(caminho)
	texto := string(dados)
	for _, deve := range []string{`antiga = "v1"`, `nova = "v2"`, "# minhas refs", "# azul da marca", `fundo = "#0000ff"`} {
		if !strings.Contains(texto, deve) {
			t.Errorf("faltou %q em:\n%s", deve, texto)
		}
	}
	if strings.Contains(texto, `"a"`) || strings.Count(texto, "antiga") != 1 {
		t.Errorf("refs antigas não foram trocadas:\n%s", texto)
	}
}
