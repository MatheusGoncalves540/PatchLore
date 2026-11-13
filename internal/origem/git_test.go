package origem

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// repoDeTeste monta: main (c1, tag v1) → dev (c2) → feat (c3) mesclada em dev
// com --no-ff, para provar que o merge fica de fora.
func repoDeTeste(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Teste")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "teste@exemplo.invalid")
	}
	rodar := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if saida, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, saida)
		}
	}
	commit := func(arquivo, msg string) {
		if err := os.WriteFile(filepath.Join(dir, arquivo), []byte(msg), 0o644); err != nil {
			t.Fatal(err)
		}
		rodar("add", arquivo)
		rodar("commit", "-q", "-m", msg)
	}
	rodar("init", "-q", "-b", "main")
	commit("base.txt", "c1: base")
	rodar("tag", "v1")
	rodar("switch", "-q", "-c", "dev")
	commit("a.txt", "c2: busca")
	rodar("switch", "-q", "-c", "feat")
	commit("b.txt", "c3: exportar")
	rodar("switch", "-q", "dev")
	rodar("merge", "-q", "--no-ff", "-m", "Merge feat", "feat")
	return dir
}

func TestLocal(t *testing.T) {
	dir := repoDeTeste(t)
	m, err := Local(dir, "main", "dev")
	if err != nil {
		t.Fatal(err)
	}
	// Os commits nascem no mesmo segundo, então a ordem entre eles empata.
	if slices.Sort(m.Commits); !slices.Equal(m.Commits, []string{"c2: busca", "c3: exportar"}) {
		t.Fatalf("commits sem o merge: %q", m.Commits)
	}
	if len(m.Arquivos) != 2 || m.Arquivos[0].Nome != "a.txt" || m.Arquivos[0].Adicoes != 1 {
		t.Fatalf("arquivos: %+v", m.Arquivos)
	}
	if m.Repo != filepath.Base(dir) || m.Antiga != "main" || m.Nova != "dev" {
		t.Fatalf("metadados: %+v", m)
	}
}

func TestLocalSemCommits(t *testing.T) {
	if _, err := Local(repoDeTeste(t), "dev", "main"); !errors.Is(err, ErrSemCommits) {
		t.Fatalf("esperava ErrSemCommits, veio %v", err)
	}
}

func TestLocalForaDeRepo(t *testing.T) {
	if _, err := Local(t.TempDir(), "a", "b"); err == nil {
		t.Fatal("esperava erro fora de um repositório")
	}
}

func TestRefs(t *testing.T) {
	refs, err := Refs(repoDeTeste(t))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(refs, []string{"dev", "feat", "main", "v1"}) {
		t.Fatalf("refs: %v", refs)
	}
}
