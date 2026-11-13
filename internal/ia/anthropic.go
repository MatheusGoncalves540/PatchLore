package ia

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

const versaoAnthropic = "2026-10-21"

func cabecalhosAnthropic(p config.Provedor) map[string]string {
	return map[string]string{"x-api-key": p.Chave, "anthropic-version": versaoAnthropic}
}

// oficial diz se é a API da própria Anthropic. Proxies compatíveis costumam
// rejeitar campos novos, então recursos beta só vão para a oficial.
func oficial(p config.Provedor) bool {
	u, err := url.Parse(p.URL)
	return err == nil && u.Hostname() == "api.anthropic.com"
}

type respostaAnthropic struct {
	StopReason  string `json:"stop_reason"`
	StopDetails *struct {
		Category string `json:"category"`
	} `json:"stop_details"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func viaAnthropic(ctx context.Context, p config.Provedor, m notas.Mudancas, t tema.Tema) (notas.Notas, error) {
	corpo := map[string]any{
		"model":      p.Modelo,
		"max_tokens": 16000,
		"system":     Sistema(t),
		"messages":   []map[string]any{{"role": "user", "content": Prompt(m)}},
		// Saída estruturada: a API só deixa o modelo responder dentro do schema.
		"output_config": map[string]any{"format": map[string]any{"type": "json_schema", "schema": Schema(t)}},
	}
	cabecalhos := cabecalhosAnthropic(p)
	if oficial(p) {
		// Se o modelo recusar por engano, a própria API tenta de novo num
		// modelo reserva, em vez de devolver o patchnotes vazio.
		cabecalhos["anthropic-beta"] = "server-side-fallback-2026-07-01"
		corpo["fallbacks"] = "default"
	}

	var r respostaAnthropic
	if err := enviar(ctx, http.MethodPost, semBarra(p.URL)+"/v1/messages", cabecalhos, corpo, &r); err != nil {
		return notas.Notas{}, err
	}
	if r.StopReason == "refusal" {
		categoria := "sem categoria"
		if r.StopDetails != nil && r.StopDetails.Category != "" {
			categoria = r.StopDetails.Category
		}
		return notas.Notas{}, fmt.Errorf("o modelo recusou o pedido (%s)", categoria)
	}
	// A resposta pode trazer blocos de raciocínio antes; o JSON vem no texto.
	for _, bloco := range r.Content {
		if bloco.Type == "text" {
			n, err := decodificar(bloco.Text, t)
			if err != nil {
				return n, fmt.Errorf("resposta fora do formato (stop_reason=%s): %w", r.StopReason, err)
			}
			return n, nil
		}
	}
	return notas.Notas{}, fmt.Errorf("resposta sem texto (stop_reason=%s)", r.StopReason)
}
