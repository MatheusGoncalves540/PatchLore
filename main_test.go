package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// repo cria main (1 commit) e dev (2 commits a mais) num diretório temporário.
func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Teste")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "teste@exemplo.invalid")
	}
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(arquivo, msg string) {
		os.WriteFile(filepath.Join(dir, arquivo), []byte(msg), 0o644)
		git("add", arquivo)
		git("commit", "-q", "-m", msg)
	}
	git("init", "-q", "-b", "main")
	commit("base.txt", "base")
	git("switch", "-q", "-c", "dev")
	commit("busca.txt", "feat: busca de clientes")
	commit("cpf.txt", "feat: CPF obrigatório")
	return dir
}

// iaFalsa responde como uma API compatível com OpenAI e conta as chamadas.
func iaFalsa(t *testing.T, chamadas *atomic.Int32) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamadas.Add(1)
		corpo, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(corpo), "feat: busca de clientes") {
			t.Errorf("o prompt não trouxe os commits: %s", corpo)
		}
		notas := `{"titulo":"Busca e CPF","itens":[{"tag":"novo","titulo":"Busca","texto":"Busque clientes pelo nome."},{"tag":"regra","titulo":"CPF","texto":"CPF agora é obrigatório."}]}`
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": notas}}}})
		w.Write(resp)
	}))
	t.Cleanup(s.Close)
	return s.URL
}

func TestPontaAPonta(t *testing.T) {
	dir := repo(t)
	var chamadas atomic.Int32
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATCHLORE_API_KEY", "sk-teste")
	t.Setenv("PATCHLORE_PROVIDER", "falsa")
	t.Setenv("PATCHLORE_API_TYPE", "openai")
	t.Setenv("PATCHLORE_BASE_URL", iaFalsa(t, &chamadas))
	t.Setenv("PATCHLORE_MODEL", "modelo-falso")
	t.Chdir(dir)

	saida := filepath.Join(t.TempDir(), "out")
	if err := rodar([]string{"main..dev", "-o", saida}, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, nome := range []string{"notes.json", "patchnotes.png", "patchnotes.pdf", "patchnotes.md"} {
		if info, err := os.Stat(filepath.Join(saida, nome)); err != nil || info.Size() == 0 {
			t.Errorf("%s não foi gerado: %v", nome, err)
		}
	}
	md, _ := os.ReadFile(filepath.Join(saida, "patchnotes.md"))
	if !strings.Contains(string(md), "**REGRA · CPF**") || !strings.Contains(string(md), "2 commits") {
		t.Errorf("markdown inesperado:\n%s", md)
	}

	// Sem argumentos e sem terminal, reaproveita as refs salvas no projeto.
	if err := rodar([]string{"-o", saida}, io.Discard); err != nil {
		t.Fatalf("segunda execução sem refs: %v", err)
	}
	if chamadas.Load() != 2 {
		t.Errorf("esperava 2 chamadas à IA, houve %d", chamadas.Load())
	}

	// Sem commits novos, avisa e não chama a IA.
	if err := rodar([]string{"dev..main", "-o", saida}, io.Discard); err == nil || !strings.Contains(err.Error(), "nenhum commit") {
		t.Errorf("esperava aviso de zero commits: %v", err)
	}
	if chamadas.Load() != 2 {
		t.Error("não deveria chamar a IA sem commits")
	}
}

func TestForaDeRepo(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := rodar([]string{"a..b"}, io.Discard); err == nil || !strings.Contains(err.Error(), "repositório git") {
		t.Fatalf("esperava erro claro fora de repo: %v", err)
	}
}
