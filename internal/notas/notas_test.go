package notas

import (
	"os"
	"strings"
	"testing"

	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

func temaPadrao(t *testing.T) tema.Tema {
	t.Helper()
	tm, err := tema.Carregar("", "")
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func item(tag string) Item { return Item{Tag: tag, Titulo: "t", Texto: "x"} }

func TestValidar(t *testing.T) {
	tm := temaPadrao(t)
	casos := []struct {
		nome  string
		notas Notas
		erro  string
	}{
		{"ok", Notas{"t", []Item{item("novo")}}, ""},
		{"tag inválida", Notas{"t", []Item{item("feature")}}, `"feature" não existe`},
		{"zero itens", Notas{"t", nil}, "itens: 0"},
		{"acima do limite", Notas{"t", []Item{item("novo"), item("novo"), item("novo"), item("novo"), item("novo"), item("novo"), item("novo")}}, "itens: 7"},
		{"título vazio", Notas{" ", []Item{item("novo")}}, "titulo vazio"},
		{"texto vazio", Notas{"t", []Item{{Tag: "regra", Titulo: "a"}}}, "itens[0].texto vazio"},
	}
	for _, c := range casos {
		err := Validar(c.notas, tm)
		if c.erro == "" && err != nil {
			t.Errorf("%s: erro inesperado %v", c.nome, err)
		}
		if c.erro != "" && (err == nil || !strings.Contains(err.Error(), c.erro)) {
			t.Errorf("%s: esperava %q, veio %v", c.nome, c.erro, err)
		}
	}
}

func TestExemploValido(t *testing.T) {
	dados, err := os.ReadFile("../../testdata/notes.json")
	if err != nil {
		t.Fatal(err)
	}
	n, err := Decodificar(dados, temaPadrao(t))
	if err != nil || len(n.Itens) != 6 {
		t.Fatalf("testdata/notes.json: %v (%d itens)", err, len(n.Itens))
	}
}
