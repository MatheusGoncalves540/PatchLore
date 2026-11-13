package ia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
	"github.com/MatheusGoncalves540/patchlore/internal/notas"
	"github.com/MatheusGoncalves540/patchlore/internal/tema"
)

func temaPadrao(t *testing.T) tema.Tema {
	t.Helper()
	tm, err := tema.Carregar("", "")
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

var mudancas = notas.Mudancas{
	Repo: "app", Antiga: "main", Nova: "dev", Commits: []string{"feat: busca"},
	Arquivos: []notas.Arquivo{{Nome: "a.ts", Status: "modified", Adicoes: 1}},
}

const notasOK = `{"titulo":"t","itens":[{"tag":"regra","titulo":"x","texto":"y"}]}`

// servidor responde sempre o mesmo corpo e guarda a última requisição.
func servidor(t *testing.T, resposta string, visto *map[string]any, cabecalhos *http.Header) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if visto != nil {
			json.NewDecoder(r.Body).Decode(visto)
		}
		if cabecalhos != nil {
			*cabecalhos = r.Header.Clone()
		}
		w.Write([]byte(resposta))
	}))
	t.Cleanup(s.Close)
	return s.URL
}

func respostaTexto(stop, texto string) string {
	b, _ := json.Marshal(map[string]any{
		"stop_reason": stop,
		"content":     []map[string]string{{"type": "thinking", "thinking": ""}, {"type": "text", "text": texto}},
	})
	return string(b)
}

func TestAnthropicValida(t *testing.T) {
	var corpo map[string]any
	var cab http.Header
	url := servidor(t, respostaTexto("end_turn", notasOK), &corpo, &cab)
	p := config.Provedor{Tipo: config.TipoAnthropic, URL: url + "/", Chave: "sk", Modelo: "m"}

	n, err := Resumir(context.Background(), p, mudancas, temaPadrao(t))
	if err != nil || n.Itens[0].Tag != "regra" {
		t.Fatalf("notas: %+v %v", n, err)
	}
	if cab.Get("x-api-key") != "sk" || cab.Get("anthropic-version") == "" {
		t.Errorf("cabeçalhos: %v", cab)
	}
	if _, ok := corpo["fallbacks"]; ok {
		t.Error("fallbacks só deve ir para a API oficial")
	}
	formato := corpo["output_config"].(map[string]any)["format"].(map[string]any)
	if formato["type"] != "json_schema" {
		t.Errorf("formato: %v", formato)
	}
	if strings.Contains(mustJSON(formato), "minItems") {
		t.Error("schema não pode levar minItems/maxItems")
	}
}

func TestAnthropicRecusa(t *testing.T) {
	url := servidor(t, `{"stop_reason":"refusal","stop_details":{"category":"cyber"},"content":[]}`, nil, nil)
	_, err := Resumir(context.Background(), config.Provedor{Tipo: config.TipoAnthropic, URL: url}, mudancas, temaPadrao(t))
	if err == nil || !strings.Contains(err.Error(), "recusou") || !strings.Contains(err.Error(), "cyber") {
		t.Fatalf("recusa: %v", err)
	}
}

func TestAnthropicForaDoFormato(t *testing.T) {
	url := servidor(t, respostaTexto("end_turn", `{"titulo":"t","itens":[{"tag":"feature","titulo":"x","texto":"y"}]}`), nil, nil)
	_, err := Resumir(context.Background(), config.Provedor{Tipo: config.TipoAnthropic, URL: url}, mudancas, temaPadrao(t))
	if err == nil || !strings.Contains(err.Error(), "fora do formato") {
		t.Fatalf("formato: %v", err)
	}
}

func TestPromptESistema(t *testing.T) {
	p := Prompt(mudancas)
	for _, deve := range []string{"- feat: busca", "modified +1 -0 a.ts", "main → dev"} {
		if !strings.Contains(p, deve) {
			t.Errorf("prompt sem %q:\n%s", deve, p)
		}
	}
	if s := Sistema(temaPadrao(t)); !strings.Contains(s, "regra = nova validação") {
		t.Errorf("sistema sem as tags do tema:\n%s", s)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
