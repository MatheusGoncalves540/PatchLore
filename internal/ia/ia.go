// Pacote ia conversa com o modelo. Tudo o que ele recebe (prompt do sistema,
// tags válidas, schema) sai do tema, então trocar uma tag no tema já muda o
// que a IA pode responder, sem tocar em código.
package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

// Modelos que pensam antes de responder podem levar minutos; o timeout
// padrão de clientes HTTP cortaria a resposta no meio.
var cliente = &http.Client{Timeout: 10 * time.Minute}

// Resumir pede à IA o texto do patchnotes e só devolve notas já validadas.
func Resumir(ctx context.Context, p config.Provedor, m notas.Mudancas, t tema.Tema) (notas.Notas, error) {
	switch p.Tipo {
	case config.TipoAnthropic:
		return viaAnthropic(ctx, p, m, t)
	case config.TipoOpenAI:
		return viaOpenAI(ctx, p, m, t)
	default:
		return notas.Notas{}, fmt.Errorf("formato de API desconhecido: %q", p.Tipo)
	}
}

func Sistema(t tema.Tema) string {
	var tags []string
	for _, nome := range t.Nomes() {
		tags = append(tags, nome+" = "+t.Tags[nome].Descricao)
	}
	return strings.TrimSpace(t.Prompt) + "\nTags: " + strings.Join(tags, "; ") + "."
}

// Prompt descreve as mudanças do jeito que uma pessoa leria: cabeçalho,
// descrição e os commits que podem ter efeito visível. Nomes de arquivo e
// contagem de linhas ficam de fora: são a principal fonte de texto técnico
// que não interessa a quem usa o sistema.
func Prompt(m notas.Mudancas) string {
	cabecalho := "# " + m.Repo
	if m.Numero > 0 {
		cabecalho = fmt.Sprintf("# PR #%d", m.Numero)
	}
	if m.Titulo != "" {
		cabecalho += ": " + m.Titulo
	}
	partes := []string{cabecalho, m.Antiga + " → " + m.Nova}
	if d := strings.TrimSpace(m.Descricao); d != "" {
		partes = append(partes, "## Descrição\n"+d)
	}
	var commits strings.Builder
	for _, c := range m.Commits {
		msg := limpar(c)
		if util(msg) {
			commits.WriteString("- " + msg + "\n")
		}
	}
	if lista := strings.TrimSpace(commits.String()); lista != "" {
		partes = append(partes, "## Commits\n"+lista)
	}
	return strings.Join(partes, "\n\n")
}

// ruido casa assuntos que não descrevem nada visível: merges e commits de
// manutenção. A instrução do tema também manda ignorá-los, mas tirá-los antes
// evita que o modelo se apoie em detalhes de bastidor.
var ruido = regexp.MustCompile(`(?i)^(merge\b|bump\b|release\b|(chore|docs?|tests?|ci|build|deps|lint|style|refactor|wip|fixup|squash)(\([^)]*\))?!?:)`)

// trailer casa as linhas que o git acrescenta ao fim da mensagem.
var trailer = regexp.MustCompile(`(?i)^(signed-off-by|co-authored-by|reviewed-by|change-id|refs?|see-also)\s*:`)

// util diz se o commit pode descrever algo que a pessoa vê.
func util(msg string) bool {
	assunto := strings.TrimSpace(strings.SplitN(msg, "\n", 2)[0])
	return assunto != "" && !ruido.MatchString(assunto)
}

// limpar tira os trailers de revisão, que só interessam ao git.
func limpar(msg string) string {
	linhas := strings.Split(strings.ReplaceAll(msg, "\r\n", "\n"), "\n")
	uteis := make([]string, 0, len(linhas))
	for _, l := range linhas {
		if trailer.MatchString(strings.TrimSpace(l)) {
			continue
		}
		uteis = append(uteis, strings.TrimRight(l, " \t"))
	}
	return strings.TrimSpace(strings.Join(uteis, "\n"))
}

// Schema descreve as notas para a IA. Fica enxuto de propósito: a API da
// Anthropic recusa limites como minItems/maxItems, então eles vão como texto
// na descrição e quem garante de verdade é notas.Validar.
func Schema(t tema.Tema) map[string]any {
	texto := func(descricao string) map[string]any {
		return map[string]any{"type": "string", "description": descricao}
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"titulo", "itens"},
		"properties": map[string]any{
			"titulo": texto("Frase curta que resume a entrega para o usuário, sem ponto final e sem termos técnicos"),
			"itens": map[string]any{
				"type":        "array",
				"description": fmt.Sprintf("De 1 a %d itens, do mais importante para o menos", t.LimiteItens),
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"tag", "titulo", "texto"},
					"properties": map[string]any{
						"tag":    map[string]any{"type": "string", "enum": t.Nomes()},
						"titulo": texto("2 a 4 palavras, usando o nome que aparece na tela (ex.: Filtro de clientes)"),
						"texto":  texto("1 ou 2 frases, até ~240 caracteres, sobre o que a pessoa passa a poder fazer ou vê diferente; sem nomes de arquivo, código ou detalhes de implementação"),
					},
				},
			},
		},
	}
}

// decodificar aceita a resposta crua do modelo, inclusive embrulhada em
// cerca ```json, que modelos sem saída estruturada adoram mandar.
func decodificar(texto string, t tema.Tema) (notas.Notas, error) {
	texto = strings.TrimSpace(texto)
	texto = strings.TrimPrefix(texto, "```json")
	texto = strings.TrimPrefix(texto, "```")
	texto = strings.TrimSuffix(texto, "```")
	return notas.Decodificar([]byte(strings.TrimSpace(texto)), t)
}

// enviar faz a requisição e decodifica o JSON. Em erro, devolve o começo do
// corpo: é ali que as APIs explicam o que deu errado (chave, modelo, cota).
func enviar(ctx context.Context, metodo, url string, cabecalhos map[string]string, corpo, destino any) error {
	var leitor io.Reader
	if corpo != nil {
		dados, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(dados)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, url, leitor)
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	for k, v := range cabecalhos {
		req.Header.Set(k, v)
	}
	resp, err := cliente.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dados, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%d em %s: %.300s", resp.StatusCode, url, dados)
	}
	if err := json.Unmarshal(dados, destino); err != nil {
		return fmt.Errorf("resposta inválida de %s: %w", url, err)
	}
	return nil
}

func semBarra(url string) string { return strings.TrimRight(url, "/") }
