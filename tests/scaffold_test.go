// Package tests は足場の不変量(TEST_SPEC L0)を検査する。
package tests

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot はテストの作業ディレクトリ(tests/)の親。
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(wd)
}

// T-002 / N-01: go.mod に require が無い(標準ライブラリのみ)。
func TestGoModHasNoRequire(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "require") {
			t.Fatalf("go.mod に require がある: %q", line)
		}
	}
}

// T-003 / N-02: internal/ 配下の Go ファイルは syscall/js を import しない。
// internal/ に .go ファイルが 1 つも無い場合も失敗にする(走査対象が消えたことに気づくため)。
func TestInternalDoesNotImportSyscallJS(t *testing.T) {
	root := filepath.Join(repoRoot(t), "internal")
	fset := token.NewFileSet()
	n := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		n++
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "syscall/js" {
				t.Errorf("%s が syscall/js を import している(N-02)", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("internal/ を走査できない: %v", err)
	}
	if n == 0 {
		t.Fatal("internal/ に Go ファイルが無い")
	}
}
