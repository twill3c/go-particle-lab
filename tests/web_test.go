package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// T-302 / N-03: web/app.js は粒子の運動則を持たない。
// 数値積分(`x += v * dt` 型)・距離計算(hypot / sqrt)が無いことを検査する。
func TestAppJSHasNoPhysics(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, pat := range []string{`\+=\s*[\w.]+\s*\*\s*dt\b`, `Math\.hypot`, `Math\.sqrt`} {
		if re := regexp.MustCompile(pat); re.MatchString(src) {
			t.Errorf("app.js に運動則らしきコード %q がある(N-03): %q", pat, re.FindString(src))
		}
	}
	if !strings.Contains(src, "GoParticleLab") {
		t.Error("app.js が Wasm ブリッジ GoParticleLab を参照していない")
	}
}

// F-46: index.html にフリート共通フッタ 5 項目がある(fleet-footer-standard)。
func TestFooterHasFiveLinks(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	i := strings.Index(src, "<footer")
	j := strings.Index(src, "</footer>")
	if i < 0 || j < i {
		t.Fatal("footer が無い")
	}
	footer := src[i:j]
	for _, want := range []string{"LICENSE", "github.com/twill3c/go-particle-lab", "claude.ai/code/artifact", "app-menu-amber.vercel.app"} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer に %q が無い", want)
		}
	}
	if n := strings.Count(footer, "<a "); n != 5 {
		t.Errorf("footer のリンク数 %d (期待 5)", n)
	}
}
