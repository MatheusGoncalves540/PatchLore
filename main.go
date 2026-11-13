// O patchlore transforma o histórico entre duas refs de um repositório git em
// patchnotes visuais. O fluxo é sempre o mesmo: ler as mudanças (commits e
// arquivos), pedir à IA só o texto num JSON de formato fixo, validar esse JSON
// contra o tema e desenhar o resultado em PNG, PDF e Markdown. A IA nunca toca
// no visual; é isso que garante que todo patchnotes saia com a mesma cara.
package main

import (
	"flag"
	"fmt"
	"os"
)

const uso = `patchlore: patchnotes visuais a partir do histórico do git

Uso
  patchlore antiga..nova    gera o patchnotes das mudanças entre duas refs
  patchlore                 escolhe as refs num menu (ou reaproveita as do .patchlore.toml)
  patchlore config          configura provedores de IA (Claude, DeepSeek, outros)

Opções
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), uso)
		flag.PrintDefaults()
	}
	flag.Parse()
	os.Exit(0)
}
