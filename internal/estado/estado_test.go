package estado

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCadaRepoGuardaOSepu(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")

	if e, err := Ler(a); err != nil || e.Commit != "" {
		t.Fatalf("sem arquivo deveria vir vazio: %+v %v", e, err)
	}
	if err := Salvar(a, Estado{Ref: "development", Commit: "1111111"}); err != nil {
		t.Fatal(err)
	}
	if err := Salvar(b, Estado{Ref: "production", Commit: "2222222"}); err != nil {
		t.Fatal(err)
	}
	if e, _ := Ler(a); e != (Estado{Ref: "development", Commit: "1111111"}) {
		t.Fatalf("repo a: %+v", e)
	}
	if e, _ := Ler(b); e != (Estado{Ref: "production", Commit: "2222222"}) {
		t.Fatalf("repo b: %+v", e)
	}
	if _, err := os.Stat(filepath.Join(a, arquivo)); !os.IsNotExist(err) {
		t.Error("o estado não pode ser gravado dentro do repositório")
	}
}

func TestSalvarPreservaOsOutros(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	for _, par := range []struct {
		raiz string
		e    Estado
	}{{a, Estado{Ref: "main", Commit: "1111111"}}, {b, Estado{Ref: "dev", Commit: "2222222"}}} {
		if err := Salvar(par.raiz, par.e); err != nil {
			t.Fatal(err)
		}
	}
	if e, _ := Ler(a); e.Commit != "1111111" {
		t.Fatalf("a foi perdido: %+v", e)
	}
}

func TestArquivoInvalido(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "patchlore"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Caminho(), []byte("repos = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Ler("."); err == nil {
		t.Fatal("esperava erro de TOML inválido")
	}
}
