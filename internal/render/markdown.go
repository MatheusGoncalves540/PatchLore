package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

// Markdown é a versão em texto do patchnotes, para colar em changelog,
// comentário de PR ou chat.
func Markdown(n notas.Notas, m notas.Mudancas, t tema.Tema) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### O que mudou: %s\n\n", n.Titulo)
	for _, it := range n.Itens {
		fmt.Fprintf(&b, "- **%s · %s**: %s\n", t.Tags[it.Tag].Rotulo, it.Titulo, it.Texto)
	}
	fmt.Fprintf(&b, "\n<sub>%s · %s · %s</sub>\n", Rotulo(m), Refs(m), Contagem(m))
	return b.String()
}

// NotasJSON é o notes.json salvo ao lado das imagens: quem quiser corrigir o
// texto edita esse arquivo e roda de novo com -n, sem pagar outra chamada à IA.
func NotasJSON(n notas.Notas) []byte {
	dados, _ := json.MarshalIndent(n, "", "  ")
	return append(dados, '\n')
}
