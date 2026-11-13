// Pacote render desenha o patchnotes. O layout é montado uma vez, em pixels
// CSS, e o mesmo desenho vira PNG e PDF: assim os dois nunca divergem.
// As medidas copiam o template HTML da versão Node, para o visual não mudar.
package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	"image/png"
	"regexp"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/pdf"
	"github.com/tdewolff/canvas/renderers/rasterizer"

	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

var (
	//go:embed fontes/Inter-Regular.ttf
	interRegular []byte
	//go:embed fontes/Inter-SemiBold.ttf
	interSemiBold []byte
	//go:embed fontes/Inter-Bold.ttf
	interBold []byte
)

// Medidas em px CSS, as mesmas do template HTML antigo.
const (
	largura     = 960.0
	margem      = 56.0
	espaco      = 32.0 // entre cabeçalho, grade e rodapé
	espacoGrade = 20.0
	paddingCard = 28.0
	espacoCard  = 12.0
	raioCard    = 16.0
	ladoIcone   = 32.0
	escalaPNG   = 2.0 // PNG em 2x: 960 px CSS viram 1920 px
	// 1 px CSS = 1 pt no PDF; o canvas trabalha em milímetros.
	mmPorPt = 25.4 / 72
)

// Imagens devolve o PNG e o PDF do patchnotes.
func Imagens(n notas.Notas, m notas.Mudancas, t tema.Tema) (pngBytes, pdfBytes []byte, err error) {
	c, err := desenhar(n, m, t)
	if err != nil {
		return nil, nil, err
	}
	var bufPNG, bufPDF bytes.Buffer
	img := rasterizer.Draw(c, canvas.DPMM(escalaPNG/mmPorPt), canvas.DefaultColorSpace)
	if err := png.Encode(&bufPNG, img); err != nil {
		return nil, nil, err
	}
	p := pdf.New(&bufPDF, c.W, c.H, nil)
	c.RenderTo(p)
	if err := p.Close(); err != nil {
		return nil, nil, err
	}
	return bufPNG.Bytes(), bufPDF.Bytes(), nil
}

// Refs é o texto "antiga → nova" usado na pílula e no rodapé.
func Refs(m notas.Mudancas) string { return m.Antiga + " → " + m.Nova }

// Rotulo do topo: nome do repo e, quando vier de uma PR, o número dela.
func Rotulo(m notas.Mudancas) string {
	if m.Numero > 0 {
		return fmt.Sprintf("%s · PR #%d", m.Repo, m.Numero)
	}
	return m.Repo
}

// Contagem do rodapé, igual à da versão Node.
func Contagem(m notas.Mudancas) string {
	return fmt.Sprintf("%d commits · %d arquivos", len(m.Commits), len(m.Arquivos))
}

func desenhar(n notas.Notas, m notas.Mudancas, t tema.Tema) (*canvas.Canvas, error) {
	cores, err := novaPaleta(t)
	if err != nil {
		return nil, err
	}
	f, err := carregarFontes()
	if err != nil {
		return nil, err
	}
	l := novoLayout(n, m, cores, f)

	c := canvas.New(largura*mmPorPt, l.altura*mmPorPt)
	ctx := canvas.NewContext(c)
	// Origem no canto de cima e y crescendo para baixo, como no CSS; e a
	// escala faz cada unidade do layout valer 1 px CSS (1 pt).
	ctx.SetCoordSystem(canvas.CartesianIV)
	ctx.SetView(canvas.Identity.Scale(mmPorPt, mmPorPt))
	l.desenhar(ctx)
	return c, nil
}

// layout guarda o que foi medido antes de desenhar: a altura da imagem
// depende do texto, então medimos tudo primeiro e só depois criamos o canvas.
type layout struct {
	cores  paleta
	f      fontes
	n      notas.Notas
	m      notas.Mudancas
	altura float64

	rotulo, h1, sub, pill, rodape fonte
	linhasSub                     []string
	alturaCabecalho               float64
	pillW, pillH                  float64
	cards                         []card
	linhasGrade                   []float64 // altura de cada linha da grade
	topoGrade, topoRodape         float64
}

type card struct {
	tag                 corTag
	titulo, texto       []string
	altura              float64
	tagW, tagH          float64
}

const larguraCard = (largura - 2*margem - espacoGrade) / 2

func novoLayout(n notas.Notas, m notas.Mudancas, cores paleta, f fontes) *layout {
	l := &layout{cores: cores, f: f, n: n, m: m}
	l.rotulo = f.nova(13, cores.apagado, canvas.FontSemiBold, 0)
	l.h1 = f.nova(48, cores.texto, canvas.FontBold, 48*1.05)
	l.sub = f.nova(17, cores.suave, canvas.FontRegular, 0)
	l.pill = f.nova(13, cores.suave, canvas.FontSemiBold, 0)
	l.rodape = f.nova(13, cores.apagado, canvas.FontRegular, 0)

	// Cabeçalho: rótulo, "O que mudou" (margem de 10 em cima e embaixo) e o
	// título das notas; a pílula das refs fica à direita, alinhada embaixo.
	l.pillW, l.pillH = l.pill.face.TextWidth(Refs(m))+28, l.pill.alturaLinh+16+2
	l.linhasSub = l.sub.quebrar(n.Titulo, largura-2*margem-l.pillW-24)
	// O rótulo é um span solto, então a linha dele tem a altura da fonte base (16px).
	alturaRotulo := f.nova(16, cores.texto, canvas.FontRegular, 0).alturaLinh
	l.alturaCabecalho = alturaRotulo + 10 + l.h1.alturaLinh + 10 + l.sub.altura(l.linhasSub)

	titulo := f.nova(22, cores.texto, canvas.FontBold, 0)
	texto := f.nova(15, cores.suave, canvas.FontRegular, 15*1.5)
	tag := f.nova(12, cores.texto, canvas.FontSemiBold, 0)
	larguraTexto := larguraCard - 2 - 2*paddingCard
	for _, it := range n.Itens {
		c := card{tag: cores.tags[it.Tag]}
		c.titulo = titulo.quebrar(it.Titulo, larguraTexto)
		c.texto = texto.quebrar(it.Texto, larguraTexto)
		c.tagW, c.tagH = tag.face.TextWidth(c.tag.rotulo)+20, tag.alturaLinh+8
		c.altura = 2 + 2*paddingCard + ladoIcone + espacoCard + titulo.altura(c.titulo) + espacoCard + texto.altura(c.texto)
		l.cards = append(l.cards, c)
	}
	// Na grade, os dois cards de uma linha ficam com a altura do maior.
	for i := 0; i < len(l.cards); i += 2 {
		h := l.cards[i].altura
		if i+1 < len(l.cards) {
			h = max(h, l.cards[i+1].altura)
		}
		l.linhasGrade = append(l.linhasGrade, h)
	}
	alturaGrade := espacoGrade * float64(len(l.linhasGrade)-1)
	for _, h := range l.linhasGrade {
		alturaGrade += h
	}

	l.topoGrade = margem + l.alturaCabecalho + espaco
	l.topoRodape = l.topoGrade + alturaGrade + espaco
	l.altura = l.topoRodape + 1 + 18 + l.rodape.alturaLinh + margem
	return l
}

func (l *layout) desenhar(ctx *canvas.Context) {
	f, cores := l.f, l.cores
	retangulo(ctx, 0, 0, largura, l.altura, 0, cores.fundo, color.RGBA{})

	// Cabeçalho
	y := margem
	alturaRotulo := f.nova(16, cores.texto, canvas.FontRegular, 0).alturaLinh
	l.rotulo.escrever(ctx, margem, y+(alturaRotulo-l.rotulo.alturaLinh)/2, []string{strings.ToUpper(Rotulo(l.m))}, canvas.Left)
	y += alturaRotulo + 10
	l.h1.escrever(ctx, margem, y, []string{"O que mudou"}, canvas.Left)
	y += l.h1.alturaLinh + 10
	l.sub.escrever(ctx, margem, y, l.linhasSub, canvas.Left)

	xPill := largura - margem - l.pillW
	yPill := margem + l.alturaCabecalho - l.pillH
	retangulo(ctx, xPill, yPill, l.pillW, l.pillH, l.pillH/2, cores.fundo, cores.linha)
	l.pill.escrever(ctx, xPill+14, yPill+9, []string{Refs(l.m)}, canvas.Left)

	// Grade de cards
	titulo := f.nova(22, cores.texto, canvas.FontBold, 0)
	texto := f.nova(15, cores.suave, canvas.FontRegular, 15*1.5)
	y = l.topoGrade
	for i, c := range l.cards {
		linha := i / 2
		if i%2 == 0 && i > 0 {
			y += l.linhasGrade[linha-1] + espacoGrade
		}
		x := margem + float64(i%2)*(larguraCard+espacoGrade)
		l.desenharCard(ctx, c, x, y, l.linhasGrade[linha], titulo, texto)
	}

	// Rodapé: linha divisória, título (ou as refs) à esquerda e contagem à direita.
	ctx.SetFillColor(cores.linha)
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(margem, l.topoRodape, canvas.Rectangle(largura-2*margem, 1))
	yTexto := l.topoRodape + 1 + 18
	esquerda := l.m.Titulo
	if esquerda == "" {
		esquerda = Refs(l.m)
	}
	l.rodape.escrever(ctx, margem, yTexto, []string{esquerda}, canvas.Left)
	l.rodape.escrever(ctx, largura-margem, yTexto, []string{Contagem(l.m)}, canvas.Right)
}

func (l *layout) desenharCard(ctx *canvas.Context, c card, x, y, altura float64, titulo, texto fonte) {
	retangulo(ctx, x, y, larguraCard, altura, raioCard, l.cores.card, l.cores.bordaCard)
	x += 1 + paddingCard
	y += 1 + paddingCard

	// Ícone: caminho SVG 24x24 desenhado só com traço, ampliado para 32px.
	ctx.Push()
	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(c.tag.icone)
	ctx.SetStrokeWidth(2 * ladoIcone / 24)
	ctx.SetStrokeCapper(canvas.RoundCap)
	ctx.SetStrokeJoiner(canvas.RoundJoin)
	ctx.DrawPath(x, y, c.tag.caminho.Copy().Scale(ladoIcone/24, ladoIcone/24))
	ctx.Pop()

	// Etiqueta da tag, à direita e centralizada na altura do ícone.
	xTag := x + larguraCard - 2 - 2*paddingCard - c.tagW
	yTag := y + (ladoIcone-c.tagH)/2
	retangulo(ctx, xTag, yTag, c.tagW, c.tagH, c.tagH/2, c.tag.fundo, color.RGBA{})
	tag := l.f.nova(12, c.tag.texto, canvas.FontSemiBold, 0)
	tag.escrever(ctx, xTag+10, yTag+4, []string{c.tag.rotulo}, canvas.Left)

	y += ladoIcone + espacoCard
	titulo.escrever(ctx, x, y, c.titulo, canvas.Left)
	y += titulo.altura(c.titulo) + espacoCard
	texto.escrever(ctx, x, y, c.texto, canvas.Left)
}

var hexCor = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// cor só aceita #rgb ou #rrggbb. O tema pode vir do .patchlore.toml de
// qualquer repo; melhor recusar com mensagem clara do que pintar de preto.
func cor(s string) (color.RGBA, error) {
	if !hexCor.MatchString(s) {
		return color.RGBA{}, fmt.Errorf("cor inválida no tema: %q (use #rrggbb)", s)
	}
	hex := s[1:]
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	v, _ := strconv.ParseUint(hex, 16, 32)
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, nil
}

// paleta guarda as cores e ícones do tema, já validados.
type paleta struct {
	fundo, texto, suave, apagado, card, bordaCard, linha color.RGBA
	tags                                                 map[string]corTag
}

type corTag struct {
	icone, texto, fundo color.RGBA
	caminho             *canvas.Path
	rotulo              string
}

func novaPaleta(t tema.Tema) (paleta, error) {
	var p paleta
	var erros []string
	ler := func(destino *color.RGBA, s string) {
		c, err := cor(s)
		if err != nil {
			erros = append(erros, err.Error())
		}
		*destino = c
	}
	ler(&p.fundo, t.Cores.Fundo)
	ler(&p.texto, t.Cores.Texto)
	ler(&p.suave, t.Cores.TextoSuave)
	ler(&p.apagado, t.Cores.TextoApagado)
	ler(&p.card, t.Cores.Card)
	ler(&p.bordaCard, t.Cores.BordaCard)
	ler(&p.linha, t.Cores.Linha)
	p.tags = map[string]corTag{}
	for nome, tag := range t.Tags {
		var ct corTag
		ler(&ct.icone, tag.Cor)
		ler(&ct.texto, tag.Texto)
		ler(&ct.fundo, tag.Fundo)
		// O ícone passa pelo parser de caminho SVG: só comandos de desenho
		// entram, nada de markup.
		caminho, err := canvas.ParseSVGPath(tag.Icone)
		if err != nil {
			erros = append(erros, fmt.Sprintf("ícone da tag %s: %v", nome, err))
		}
		ct.caminho, ct.rotulo = caminho, tag.Rotulo
		p.tags[nome] = ct
	}
	if len(erros) > 0 {
		return p, fmt.Errorf("tema inválido: %s", strings.Join(erros, "; "))
	}
	return p, nil
}

// fonte é uma face com a altura de linha do CSS que ela imita.
type fonte struct {
	face       *canvas.FontFace
	alturaLinh float64
}

type fontes struct{ familia *canvas.FontFamily }

func carregarFontes() (fontes, error) {
	f := canvas.NewFontFamily("Inter")
	for _, arq := range []struct {
		dados  []byte
		estilo canvas.FontStyle
	}{{interRegular, canvas.FontRegular}, {interSemiBold, canvas.FontSemiBold}, {interBold, canvas.FontBold}} {
		if err := f.LoadFont(arq.dados, 0, arq.estilo); err != nil {
			return fontes{}, err
		}
	}
	return fontes{f}, nil
}

// nova cria uma face de px CSS. O tamanho vai em pt e o canvas converte para
// mm; como desenhamos com escala mm/pt, 1 unidade do layout vale 1 px CSS.
// alturaLinha 0 imita o "line-height: normal" do navegador.
func (f fontes) nova(px float64, c color.RGBA, estilo canvas.FontStyle, alturaLinha float64) fonte {
	face := f.familia.Face(px/mmPorPt, c, estilo)
	if alturaLinha == 0 {
		m := face.Metrics()
		alturaLinha = m.Ascent + m.Descent + m.LineGap
	}
	return fonte{face, alturaLinha}
}

// quebrar divide o texto em linhas que cabem na largura, palavra por palavra,
// do mesmo jeito guloso que o navegador faz.
func (f fonte) quebrar(texto string, largura float64) []string {
	var linhas []string
	atual := ""
	for _, palavra := range strings.Fields(texto) {
		tentativa := palavra
		if atual != "" {
			tentativa = atual + " " + palavra
		}
		if atual != "" && f.face.TextWidth(tentativa) > largura {
			linhas = append(linhas, atual)
			tentativa = palavra
		}
		atual = tentativa
	}
	if atual != "" || len(linhas) == 0 {
		linhas = append(linhas, atual)
	}
	return linhas
}

func (f fonte) altura(linhas []string) float64 { return float64(len(linhas)) * f.alturaLinh }

// escrever desenha as linhas a partir do topo y, centralizando o texto em
// cada caixa de linha como o CSS faz. Com canvas.Right, x é a borda direita.
func (f fonte) escrever(ctx *canvas.Context, x, y float64, linhas []string, alinhar canvas.TextAlign) {
	m := f.face.Metrics()
	meio := (f.alturaLinh - m.Ascent - m.Descent) / 2
	for i, linha := range linhas {
		base := y + float64(i)*f.alturaLinh + meio + m.Ascent
		ctx.DrawText(x, base, canvas.NewTextLine(f.face, linha, alinhar))
	}
}

func retangulo(ctx *canvas.Context, x, y, w, h, raio float64, preenchimento, borda color.RGBA) {
	ctx.SetFillColor(preenchimento)
	ctx.SetStrokeColor(canvas.Transparent)
	if borda.A != 0 {
		ctx.SetStrokeColor(borda)
		ctx.SetStrokeWidth(1)
		// A borda do CSS fica por dentro da caixa; o traço do canvas fica no
		// meio da linha, então recuamos meio pixel para as duas baterem.
		x, y, w, h = x+0.5, y+0.5, w-1, h-1
	}
	forma := canvas.Rectangle(w, h)
	if raio > 0 {
		forma = canvas.RoundedRectangle(w, h, raio)
	}
	ctx.DrawPath(x, y, forma)
}
