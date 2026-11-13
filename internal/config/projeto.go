package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const ArquivoProjeto = ".patchlore.toml"

// Projeto é o que o repo lembra entre execuções: as refs da última vez.
// A seção [tema] do mesmo arquivo é lida pelo pacote tema.
type Projeto struct {
	Antiga string `toml:"antiga"`
	Nova   string `toml:"nova"`
}

func CaminhoProjeto(raiz string) string { return filepath.Join(raiz, ArquivoProjeto) }

func LerProjeto(caminho string) (Projeto, error) {
	var p Projeto
	_, err := toml.DecodeFile(caminho, &p)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, fmt.Errorf("%s inválido: %w", caminho, err)
	}
	return p, nil
}

var linhaRef = regexp.MustCompile(`^\s*(antiga|nova)\s*=`)

// SalvarRefs troca só as linhas antiga/nova do topo e mantém o resto do
// arquivo como a pessoa escreveu. Reescrever via encoder TOML apagaria os
// comentários e reordenaria a seção [tema] que ela ajustou à mão.
func SalvarRefs(caminho, antiga, nova string) error {
	atual, err := os.ReadFile(caminho)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	linhas := []string{"antiga = " + strconv.Quote(antiga), "nova = " + strconv.Quote(nova)}
	noTopo := true
	for _, l := range strings.Split(strings.ReplaceAll(string(atual), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			noTopo = false
		}
		if noTopo && linhaRef.MatchString(l) {
			continue
		}
		linhas = append(linhas, l)
	}
	saida := strings.TrimRight(strings.Join(linhas, "\n"), "\n") + "\n"
	return os.WriteFile(caminho, []byte(saida), 0o644)
}
