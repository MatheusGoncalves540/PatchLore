// Pacote origem transforma um lugar onde há histórico (repositório local ou,
// no plano 02, uma PR) em notas.Mudancas. Para o git local chamamos o binário
// git em vez de uma biblioteca: ele já está instalado, entende qualquer
// formato de ref e nunca diverge do que a pessoa vê no terminal.
package origem

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MatheusGoncalves540/patchlore/internal/notas"
)

// ErrSemCommits avisa que não há nada a resumir; chamar a IA seria pagar à toa.
var ErrSemCommits = errors.New("nenhum commit entre as refs")

func git(dir string, args ...string) (string, error) {
	// Config do repo não pode rodar programas: fsmonitor desligado aqui,
	// diff externo e textconv desligados no diff.
	cmd := exec.Command("git", append([]string{"-c", "core.fsmonitor=false"}, args...)...)
	cmd.Dir = dir
	var saida, erro bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erro
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("git não encontrado no PATH; instale o git para usar o modo local")
		}
		return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(erro.String()))
	}
	return saida.String(), nil
}

// Raiz devolve a pasta raiz do repositório que contém dir.
func Raiz(dir string) (string, error) {
	saida, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s não está dentro de um repositório git (%w)", dir, err)
	}
	return filepath.FromSlash(strings.TrimSpace(saida)), nil
}

// Refs lista branches locais, remotas e tags, nessa ordem, para o menu.
// Refs simbólicas (origin/HEAD) ficam de fora: só apontam para outra já listada.
func Refs(dir string) ([]string, error) {
	saida, err := git(dir, "for-each-ref", "--format=%(refname:short)\t%(symref)", "refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return nil, err
	}
	var refs []string
	for _, linha := range strings.Split(strings.TrimSpace(saida), "\n") {
		nome, simbolica, _ := strings.Cut(linha, "\t")
		if nome != "" && simbolica == "" {
			refs = append(refs, nome)
		}
	}
	return refs, nil
}

// Local lê o que entrou em nova desde antiga. Os commits vêm de antiga..nova
// (sem merges, que só repetem o que já está nos outros commits) e os arquivos
// de antiga...nova, que compara com o ponto onde as duas se separaram e assim
// ignora o que mudou só em antiga.
func Local(dir, antiga, nova string) (notas.Mudancas, error) {
	m := notas.Mudancas{Antiga: antiga, Nova: nova}
	raiz, err := Raiz(dir)
	if err != nil {
		return m, err
	}
	m.Repo = filepath.Base(raiz)

	// As refs podem vir do .patchlore.toml de um repo de terceiros. Com
	// --end-of-options, algo como "--output=arquivo" vira ref inválida em vez
	// de opção do git.
	log, err := git(dir, "log", "--no-merges", "--format=%B%x1e", "--end-of-options", antiga+".."+nova)
	if err != nil {
		return m, err
	}
	for _, msg := range strings.Split(log, "\x1e") {
		if msg = strings.TrimSpace(msg); msg != "" {
			m.Commits = append(m.Commits, msg)
		}
	}
	if len(m.Commits) == 0 {
		return m, fmt.Errorf("%w %s..%s", ErrSemCommits, antiga, nova)
	}

	numstat, err := git(dir, "diff", "--numstat", "--no-ext-diff", "--no-textconv", "--end-of-options", antiga+"..."+nova)
	if err != nil {
		return m, err
	}
	m.Arquivos = lerNumstat(numstat)
	return m, nil
}

// Cada linha é "adições<TAB>remoções<TAB>caminho"; binários vêm com "-".
func lerNumstat(saida string) []notas.Arquivo {
	var arquivos []notas.Arquivo
	for _, linha := range strings.Split(strings.TrimSpace(saida), "\n") {
		partes := strings.SplitN(linha, "\t", 3)
		if len(partes) != 3 {
			continue
		}
		adicoes, _ := strconv.Atoi(partes[0])
		remocoes, _ := strconv.Atoi(partes[1])
		arquivos = append(arquivos, notas.Arquivo{Nome: partes[2], Adicoes: adicoes, Remocoes: remocoes})
	}
	return arquivos
}
