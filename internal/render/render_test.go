package render

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

func exemplo(t *testing.T) (notas.Notas, notas.Mudancas, tema.Tema) {
	t.Helper()
	tm, err := tema.Carregar("", "")
	if err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile("../../testdata/notes.json")
	if err != nil {
		t.Fatal(err)
	}
	n, err := notas.Decodificar(dados, tm)
	if err != nil {
		t.Fatal(err)
	}
	m := notas.Mudancas{
		Repo: "app", Numero: 42, Titulo: "Busca de clientes e validação de CPF",
		Antiga: "production", Nova: "development",
		Commits: make([]string, 12), Arquivos: make([]notas.Arquivo, 31),
	}
	return n, m, tm
}

func TestImagens(t *testing.T) {
	n, m, tm := exemplo(t)
	pngBytes, pdfBytes, err := Imagens(n, m, tm)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	if w := img.Bounds().Dx(); w != 1920 {
		t.Errorf("PNG com %d px de largura, esperava 1920", w)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Error("PDF não começa com %PDF")
	}
	// Para conferir o visual: PATCHLORE_SALVAR=pasta go test ./internal/render/
	if dir := os.Getenv("PATCHLORE_SALVAR"); dir != "" {
		os.WriteFile(filepath.Join(dir, "patchnotes.png"), pngBytes, 0o644)
		os.WriteFile(filepath.Join(dir, "patchnotes.pdf"), pdfBytes, 0o644)
	}
}

func TestTextoLongoQuebraDentroDoCard(t *testing.T) {
	f, err := carregarFontes()
	if err != nil {
		t.Fatal(err)
	}
	_, _, tm := exemplo(t)
	cores, _ := novaPaleta(tm)
	fonteTexto := f.nova(15, cores.suave, 0, 15*1.5)
	larguraTexto := larguraCard - 2 - 2*paddingCard
	texto := strings.Repeat("palavra comprida ", 15)[:240]

	linhas := fonteTexto.quebrar(texto, larguraTexto)
	if len(linhas) < 3 {
		t.Fatalf("240 caracteres deveriam quebrar em várias linhas: %q", linhas)
	}
	for _, l := range linhas {
		if w := fonteTexto.face.TextWidth(l); w > larguraTexto {
			t.Errorf("linha estoura o card (%.1f > %.1f): %q", w, larguraTexto, l)
		}
	}

	// E o card cresce para caber o texto todo.
	n, m, _ := exemplo(t)
	n.Itens[0].Texto = texto
	l := novoLayout(n, m, cores, f)
	if l.cards[0].altura < 2+2*paddingCard+ladoIcone+2*espacoCard+fonteTexto.altura(linhas) {
		t.Errorf("card baixo demais para %d linhas: %.1f", len(linhas), l.cards[0].altura)
	}
}

func TestCorInvalida(t *testing.T) {
	_, m, tm := exemplo(t)
	tag := tm.Tags["novo"]
	tag.Fundo = `red" onload="x`
	tm.Tags["novo"] = tag
	if _, _, err := Imagens(notas.Notas{Titulo: "t", Itens: []notas.Item{{Tag: "novo", Titulo: "a", Texto: "b"}}}, m, tm); err == nil || !strings.Contains(err.Error(), "cor inválida") {
		t.Fatalf("esperava erro de cor: %v", err)
	}
}
