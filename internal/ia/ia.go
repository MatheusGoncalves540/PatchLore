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
// descrição, commits e arquivos tocados.
func Prompt(m notas.Mudancas) string {
	cabecalho := "# " + m.Repo
	if m.Numero > 0 {
		cabecalho = fmt.Sprintf("# PR #%d", m.Numero)
	}
	if m.Titulo != "" {
		cabecalho += ": " + m.Titulo
	}
	descricao := m.Descricao
	if descricao == "" {
		descricao = "(vazia)"
	}
	var commits, arquivos strings.Builder
	for _, c := range m.Commits {
		commits.WriteString("- " + c + "\n")
	}
	for _, a := range m.Arquivos {
		if a.Status != "" {
			arquivos.WriteString(a.Status + " ")
		}
		fmt.Fprintf(&arquivos, "+%d -%d %s\n", a.Adicoes, a.Remocoes, a.Nome)
	}
	return strings.Join([]string{
		cabecalho,
		m.Antiga + " → " + m.Nova,
		"## Descrição\n" + descricao,
		"## Commits\n" + strings.TrimSpace(commits.String()),
		"## Arquivos\n" + strings.TrimSpace(arquivos.String()),
	}, "\n\n")
}

// Schema descreve as notas para a IA. Fica enxuto de propósito: a API da
// Anthropic recusa limites como minItems/maxItems, então eles vão como texto
// na descrição e quem garante de verdade é notas.Validar.
func Schema(t tema.Tema) map[string]any {
	texto := func(descricao string) map[string]any { return map[string]any{"type": "string", "description": descricao} }
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"titulo", "itens"},
		"properties": map[string]any{
			"titulo": texto("Frase curta que resume a entrega, sem ponto final"),
			"itens": map[string]any{
				"type":        "array",
				"description": fmt.Sprintf("De 1 a %d itens, do mais importante para o menos", t.LimiteItens),
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"tag", "titulo", "texto"},
					"properties": map[string]any{
						"tag":    map[string]any{"type": "string", "enum": t.Nomes()},
						"titulo": texto("2 a 4 palavras"),
						"texto":  texto("1 ou 2 frases, até ~240 caracteres, do ponto de vista de quem usa o sistema"),
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
