package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

// Nem toda API compatível com OpenAI entende json_schema, mas quase todas
// entendem json_object. Então pedimos JSON genérico, mandamos o schema no
// prompt e deixamos notas.Validar barrar o que vier torto.
func viaOpenAI(ctx context.Context, p config.Provedor, m notas.Mudancas, t tema.Tema) (notas.Notas, error) {
	schema, _ := json.Marshal(Schema(t))
	corpo := map[string]any{
		"model":           p.Modelo,
		"max_tokens":      8000,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": Sistema(t) + "\n\nResponda APENAS com um JSON válido neste JSON Schema:\n" + string(schema)},
			{"role": "user", "content": Prompt(m)},
		},
	}
	var r struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := enviar(ctx, http.MethodPost, semBarra(p.URL)+"/chat/completions", cabecalhosOpenAI(p), corpo, &r); err != nil {
		return notas.Notas{}, err
	}
	if len(r.Choices) == 0 || r.Choices[0].Message.Content == "" {
		motivo := ""
		if len(r.Choices) > 0 {
			motivo = r.Choices[0].FinishReason
		}
		return notas.Notas{}, fmt.Errorf("resposta vazia (finish_reason=%s)", motivo)
	}
	return decodificar(r.Choices[0].Message.Content, t)
}

func cabecalhosOpenAI(p config.Provedor) map[string]string {
	return map[string]string{"authorization": "Bearer " + p.Chave}
}
