// Pacote estado guarda, na config do usuário, até onde cada repositório já foi
// processado. O arquivo fica fora do repo de propósito: o PatchLore não deve
// criar nada no projeto por causa disso (o .patchlore.toml continua sendo só
// do usuário, com as refs que ele escolheu).
package estado

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/MatheusGoncalves540/patchlore/internal/config"
)

const arquivo = "estado.toml"

// Estado é o ponto de parada de um repo: a ref usada na última execução e o
// commit em que ela estava. O commit é o que importa; a ref serve para saber
// até onde olhar na próxima vez (e para o título do patchnotes).
type Estado struct {
	Ref    string `toml:"ref"`
	Commit string `toml:"commit"`
}

type arquivoEstado struct {
	Repos map[string]Estado `toml:"repos"`
}

// Caminho é o arquivo único, fora dos repositórios, que guarda todos eles.
func Caminho() string { return filepath.Join(config.Dir(), arquivo) }

// chave normaliza o caminho do repo para servir de chave do mapa. No Windows
// o mesmo repositório pode aparecer com maiúsculas diferentes; sem isso
// viraria duas entradas.
func chave(raiz string) string {
	if abs, err := filepath.Abs(raiz); err == nil {
		raiz = abs
	}
	raiz = filepath.Clean(raiz)
	if runtime.GOOS == "windows" {
		raiz = strings.ToLower(raiz)
	}
	return raiz
}

func ler(caminho string) (arquivoEstado, error) {
	a := arquivoEstado{Repos: map[string]Estado{}}
	_, err := toml.DecodeFile(caminho, &a)
	if errors.Is(err, os.ErrNotExist) {
		return a, nil
	}
	if err != nil {
		return a, fmt.Errorf("%s inválido: %w", caminho, err)
	}
	if a.Repos == nil {
		a.Repos = map[string]Estado{}
	}
	return a, nil
}

// Ler devolve o último commit processado no repo. Zero se nunca rodou.
func Ler(raiz string) (Estado, error) {
	a, err := ler(Caminho())
	return a.Repos[chave(raiz)], err
}

// Salvar grava o ponto de parada de um repo preservando os outros.
func Salvar(raiz string, e Estado) error {
	caminho := Caminho()
	a, err := ler(caminho)
	if err != nil {
		return err
	}
	a.Repos[chave(raiz)] = e
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	if err := toml.NewEncoder(&b).Encode(a); err != nil {
		return err
	}
	return os.WriteFile(caminho, []byte(b.String()), 0o600)
}
