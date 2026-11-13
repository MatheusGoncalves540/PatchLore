package render

import (
	"strings"
	"testing"

	"github.com/MatheusGoncalves540/patchlore/internal/notas"
)

func TestMarkdown(t *testing.T) {
	_, m, tm := exemplo(t)
	n := notas.Notas{Titulo: "Busca", Itens: []notas.Item{{Tag: "regra", Titulo: `Sem "tipo"`, Texto: "a & b"}}}
	md := Markdown(n, m, tm)
	for _, deve := range []string{"### O que mudou: Busca", `- **REGRA · Sem "tipo"**: a & b`, "app · PR #42", "production → development", "12 commits · 31 arquivos"} {
		if !strings.Contains(md, deve) {
			t.Errorf("markdown sem %q:\n%s", deve, md)
		}
	}
}

func TestNotasJSONIdaEVolta(t *testing.T) {
	n, _, tm := exemplo(t)
	volta, err := notas.Decodificar(NotasJSON(n), tm)
	if err != nil || volta.Titulo != n.Titulo || len(volta.Itens) != len(n.Itens) {
		t.Fatalf("notes.json não volta igual: %v", err)
	}
}
