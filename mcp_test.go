package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCP(t *testing.T) {
	dir := repo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoJSON, _ := json.Marshal(dir)
	chamada := func(id int, args string) string {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"gerar_patchnotes","arguments":%s}}`, id, args)
	}
	notas := `{"titulo":"Busca","itens":[{"tag":"novo","titulo":"Busca","texto":"Busque clientes."}]}`
	entrada := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		chamada(3, `{"repositorio":`+string(repoJSON)+`,"antiga":"main","nova":"dev","notas":`+notas+`}`),
		chamada(4, `{"repositorio":`+string(repoJSON)+`,"antiga":"main"}`),
	}, "\n")

	var saida bytes.Buffer
	if err := servirMCP(strings.NewReader(entrada), &saida); err != nil {
		t.Fatal(err)
	}
	var respostas []map[string]any
	for dec := json.NewDecoder(&saida); dec.More(); {
		var r map[string]any
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		respostas = append(respostas, r)
	}
	if len(respostas) != 4 {
		t.Fatalf("esperava 4 respostas (a notificação não responde), vieram %d", len(respostas))
	}
	texto := func(i int) string { b, _ := json.Marshal(respostas[i]); return string(b) }

	if !strings.Contains(texto(0), `"protocolVersion":"2025-06-18"`) || !strings.Contains(texto(1), "gerar_patchnotes") {
		t.Errorf("initialize/tools/list: %s %s", texto(0), texto(1))
	}
	if strings.Contains(texto(2), `"isError":true`) || !strings.Contains(texto(2), "NOVO · Busca") {
		t.Errorf("tools/call: %s", texto(2))
	}
	if _, err := os.Stat(filepath.Join(dir, "patchlore-out", "patchnotes.png")); err != nil {
		t.Errorf("PNG deveria ir para patchlore-out dentro do repo: %v", err)
	}
	if !strings.Contains(texto(3), `"isError":true`) {
		t.Errorf("só antiga deveria dar isError: %s", texto(3))
	}
}
