// Pacote tema guarda tudo o que é aparência ou tom do patchnotes: cores, tags,
// limite de itens e o prompt do sistema. Fica fora do código para que cada
// pessoa (ou cada repo) ajuste o visual sem recompilar.
package tema

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
)

//go:embed tema.toml
var embutido string

type Cores struct {
	Fundo        string `toml:"fundo"`
	Texto        string `toml:"texto"`
	TextoSuave   string `toml:"texto_suave"`
	TextoApagado string `toml:"texto_apagado"`
	Card         string `toml:"card"`
	BordaCard    string `toml:"borda_card"`
	Linha        string `toml:"linha"`
}

// Tag é uma categoria de item. Cor pinta o ícone; Texto e Fundo pintam a pílula.
type Tag struct {
	Rotulo    string `toml:"rotulo"`
	Descricao string `toml:"descricao"`
	Cor       string `toml:"cor"`
	Texto     string `toml:"texto"`
	Fundo     string `toml:"fundo"`
	Icone     string `toml:"icone"`
}

type Tema struct {
	LimiteItens int            `toml:"limite_itens"`
	Prompt      string         `toml:"prompt"`
	Cores       Cores          `toml:"cores"`
	Tags        map[string]Tag `toml:"tags"`
}

// Carregar monta o tema em camadas: embutido, depois o arquivo do usuário,
// depois a seção [tema] do .patchlore.toml do repo. Quem vem depois só
// troca os campos que escreveu. Caminho vazio ou inexistente é pulado.
func Carregar(arquivoUsuario, arquivoRepo string) (Tema, error) {
	var t Tema
	if err := camada(&t, func(t *Tema) error { _, err := toml.Decode(embutido, t); return err }); err != nil {
		return t, fmt.Errorf("tema embutido: %w", err)
	}
	if err := camada(&t, func(t *Tema) error { return decodificarArquivo(arquivoUsuario, t) }); err != nil {
		return t, fmt.Errorf("tema em %s: %w", arquivoUsuario, err)
	}
	repo := struct {
		Tema *Tema `toml:"tema"`
	}{&t}
	if err := camada(&t, func(*Tema) error { return decodificarArquivo(arquivoRepo, &repo) }); err != nil {
		return t, fmt.Errorf("[tema] em %s: %w", arquivoRepo, err)
	}
	if t.LimiteItens < 1 || len(t.Tags) == 0 {
		return t, errors.New("tema precisa de limite_itens >= 1 e pelo menos uma tag")
	}
	return t, nil
}

// O decodificador TOML troca a tag inteira quando ela aparece de novo, e aí
// "mudar só a cor de novo" apagaria o rótulo e o ícone. Por isso guardamos as
// tags de antes e devolvemos os campos que a camada nova deixou em branco.
func camada(t *Tema, decodificar func(*Tema) error) error {
	antes := make(map[string]Tag, len(t.Tags))
	for k, v := range t.Tags {
		antes[k] = v
	}
	if err := decodificar(t); err != nil {
		return err
	}
	for nome, nova := range t.Tags {
		velha, ok := antes[nome]
		if !ok {
			continue
		}
		completar(&nova.Rotulo, velha.Rotulo)
		completar(&nova.Descricao, velha.Descricao)
		completar(&nova.Cor, velha.Cor)
		completar(&nova.Texto, velha.Texto)
		completar(&nova.Fundo, velha.Fundo)
		completar(&nova.Icone, velha.Icone)
		t.Tags[nome] = nova
	}
	return nil
}

func completar(campo *string, padrao string) {
	if *campo == "" {
		*campo = padrao
	}
}

func decodificarArquivo(caminho string, destino any) error {
	if caminho == "" {
		return nil
	}
	_, err := toml.DecodeFile(caminho, destino)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Nomes devolve as tags em ordem estável, para o schema e o prompt não
// mudarem de uma execução para outra.
func (t Tema) Nomes() []string {
	nomes := make([]string, 0, len(t.Tags))
	for n := range t.Tags {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	return nomes
}
