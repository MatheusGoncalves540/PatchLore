// Pacote menu reúne as telas interativas. Ficam isoladas aqui para que o
// resto do programa funcione igual sem terminal (CI, pipe), só por flags.
package menu

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// Cancelado sinaliza que a pessoa saiu do menu (Esc / Ctrl+C); não é falha.
var Cancelado = huh.ErrUserAborted

// EscolherRefs pergunta a ref antiga e a nova. As sugestões (geralmente as
// da última execução) já vêm marcadas, então repetir é só apertar Enter.
func EscolherRefs(refs []string, antiga, nova string) (string, string, error) {
	if len(refs) < 2 {
		return "", "", errors.New("o repositório precisa de pelo menos duas refs (branches ou tags)")
	}
	opcoes := huh.NewOptions(refs...)
	if err := huh.NewSelect[string]().Title("Ref antiga (o que já está publicado)").
		Options(opcoes...).Filtering(true).Height(12).Value(&antiga).Run(); err != nil {
		return "", "", err
	}
	err := huh.NewSelect[string]().Title("Ref nova (o que entra agora)").
		Options(opcoes...).Filtering(true).Height(12).Value(&nova).
		Validate(func(r string) error {
			if r == antiga {
				return errors.New("escolha uma ref diferente da antiga")
			}
			return nil
		}).Run()
	return antiga, nova, err
}
