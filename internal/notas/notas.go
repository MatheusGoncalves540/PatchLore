// Pacote notas define os dois lados da conversa com a IA: as Mudancas que
// entram (o histórico) e as Notas que saem (o texto do patchnotes).
package notas

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

type Arquivo struct {
	Nome     string
	Status   string
	Adicoes  int
	Remocoes int
}

// Mudancas é o que foi feito entre duas refs, venha de um repo local ou de
// uma PR. Numero só existe quando a origem é uma PR.
type Mudancas struct {
	Repo      string
	Numero    int
	Titulo    string
	Descricao string
	Antiga    string
	Nova      string
	Commits   []string
	Arquivos  []Arquivo
}

type Item struct {
	Tag    string `json:"tag"`
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
}

type Notas struct {
	Titulo string `json:"titulo"`
	Itens  []Item `json:"itens"`
}

// Validar confere as notas contra o tema. A IA já recebe um schema, mas
// modelos (e pessoas editando notes.json à mão) erram; aqui é a palavra final.
// Junta todos os problemas num erro só, para corrigir tudo de uma vez.
func Validar(n Notas, t tema.Tema) error {
	var erros []error
	if strings.TrimSpace(n.Titulo) == "" {
		erros = append(erros, errors.New("titulo vazio"))
	}
	if len(n.Itens) < 1 || len(n.Itens) > t.LimiteItens {
		erros = append(erros, fmt.Errorf("itens: %d, o tema aceita de 1 a %d", len(n.Itens), t.LimiteItens))
	}
	for i, it := range n.Itens {
		if _, ok := t.Tags[it.Tag]; !ok {
			erros = append(erros, fmt.Errorf("itens[%d].tag %q não existe no tema (use: %s)", i, it.Tag, strings.Join(t.Nomes(), ", ")))
		}
		if strings.TrimSpace(it.Titulo) == "" {
			erros = append(erros, fmt.Errorf("itens[%d].titulo vazio", i))
		}
		if strings.TrimSpace(it.Texto) == "" {
			erros = append(erros, fmt.Errorf("itens[%d].texto vazio", i))
		}
	}
	return errors.Join(erros...)
}

// Decodificar lê notas em JSON e já valida, que é como todo mundo as usa.
func Decodificar(dados []byte, t tema.Tema) (Notas, error) {
	var n Notas
	if err := json.Unmarshal(dados, &n); err != nil {
		return n, fmt.Errorf("JSON inválido: %w", err)
	}
	if err := Validar(n, t); err != nil {
		return n, fmt.Errorf("notas fora do formato: %w", err)
	}
	return n, nil
}
