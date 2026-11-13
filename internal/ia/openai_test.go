package ia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/MatheusGoncalves540/patchlore/internal/config"
)

func TestOpenAIResumeEListaModelos(t *testing.T) {
	var chat map[string]any
	var auth string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("authorization")
		switch r.URL.Path {
		case "/v1/models":
			w.Write([]byte(`{"data":[{"id":"b"},{"id":"a"}]}`))
		case "/v1/chat/completions":
			json.NewDecoder(r.Body).Decode(&chat)
			resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "```json\n" + notasOK + "\n```"}}}})
			w.Write(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()
	p := config.Provedor{Tipo: config.TipoOpenAI, URL: s.URL + "/v1/", Chave: "sk", Modelo: "a"}

	ids, err := Modelos(context.Background(), p)
	if err != nil || !slices.Equal(ids, []string{"a", "b"}) {
		t.Fatalf("modelos: %v %v", ids, err)
	}
	n, err := Resumir(context.Background(), p, mudancas, temaPadrao(t))
	if err != nil || n.Itens[0].Tag != "regra" {
		t.Fatalf("resumo com cerca ```json: %+v %v", n, err)
	}
	if auth != "Bearer sk" || chat["response_format"].(map[string]any)["type"] != "json_object" {
		t.Errorf("auth=%q corpo=%v", auth, chat)
	}
}

func TestOpenAIFormatoErrado(t *testing.T) {
	url := servidor(t, `{"choices":[{"message":{"content":"{\"titulo\":\"t\",\"itens\":[]}"}}]}`, nil, nil)
	_, err := Resumir(context.Background(), config.Provedor{Tipo: config.TipoOpenAI, URL: url}, mudancas, temaPadrao(t))
	if err == nil || !strings.Contains(err.Error(), "fora do formato") {
		t.Fatalf("esperava erro de formato: %v", err)
	}
}

func TestModelosAnthropicPaginado(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after_id") == "" {
			w.Write([]byte(`{"data":[{"id":"x"}],"has_more":true,"last_id":"x"}`))
			return
		}
		w.Write([]byte(`{"data":[{"id":"y"}],"has_more":false,"last_id":"y"}`))
	}))
	defer s.Close()
	ids, err := Modelos(context.Background(), config.Provedor{Tipo: config.TipoAnthropic, URL: s.URL})
	if err != nil || !slices.Equal(ids, []string{"x", "y"}) {
		t.Fatalf("paginação: %v %v", ids, err)
	}
}
