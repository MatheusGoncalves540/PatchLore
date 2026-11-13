package ia

import (
	"context"
	"net/http"
	"net/url"
	"sort"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
)

// Modelos lista ao vivo o que o provedor oferece, para o menu não depender de
// uma lista fixa que envelhece a cada lançamento.
func Modelos(ctx context.Context, p config.Provedor) ([]string, error) {
	if p.Tipo == config.TipoAnthropic {
		return modelosAnthropic(ctx, p)
	}
	var r struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := enviar(ctx, http.MethodGet, semBarra(p.URL)+"/models", cabecalhosOpenAI(p), nil, &r); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(r.Data))
	for _, m := range r.Data {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

// A API da Anthropic pagina a lista; seguimos o cursor até acabar.
// A ordem que ela devolve (mais novos primeiro) é a mais útil, então fica.
func modelosAnthropic(ctx context.Context, p config.Provedor) ([]string, error) {
	var ids []string
	depois := ""
	for {
		consulta := url.Values{"limit": {"1000"}}
		if depois != "" {
			consulta.Set("after_id", depois)
		}
		var r struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := enviar(ctx, http.MethodGet, semBarra(p.URL)+"/v1/models?"+consulta.Encode(), cabecalhosAnthropic(p), nil, &r); err != nil {
			return nil, err
		}
		for _, m := range r.Data {
			ids = append(ids, m.ID)
		}
		if !r.HasMore || r.LastID == "" {
			return ids, nil
		}
		depois = r.LastID
	}
}
