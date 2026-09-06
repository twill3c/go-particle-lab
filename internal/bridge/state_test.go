package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"unicode"

	"github.com/twill3c/go-particle-lab/internal/game"
)

// T-306 / F-32 (HC-190): 境界を渡す JSON の契約を生産側で固定する。
// (a) 全キーが小文字始まり(JS 側の慣習)、(b) app.js が読むキーがすべて存在する。
func TestStateJSONContract(t *testing.T) {
	cfg := game.DefaultConfig()
	cfg.Stage = 7 // 障害物・ブラックホールを含む最も豊かな状態
	g := game.New(cfg)
	g.AddWell(400, 300)
	g.Update(0.016)

	raw := Marshal(g, 1.23)
	var v map[string]any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("JSON でない: %v", err)
	}

	// (a) 再帰的に全キーを集め、大文字始まりを拒む。
	keys := map[string]bool{}
	var walk func(prefix string, x any)
	walk = func(prefix string, x any) {
		switch m := x.(type) {
		case map[string]any:
			for k, val := range m {
				if k == "" || !unicode.IsLower(rune(k[0])) {
					t.Errorf("キー %q が小文字始まりでない(json タグ漏れ)", prefix+k)
				}
				keys[prefix+k] = true
				walk(prefix+k+".", val)
			}
		case []any:
			for _, val := range m {
				walk(prefix, val)
			}
		}
	}
	walk("", v)

	// (b) app.js が状態オブジェクトから読むキー(s.xxx / w.xxx / o.xxx / g.xxx / b.xxx)を実物から拾う。
	src, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	prefixes := map[string]string{"s": "", "w": "wells.", "o": "obstacles.", "g": "goal.", "b": "blackHole."}
	re := regexp.MustCompile(`\b([swogb])\.([a-zA-Z]+)\b`)
	found := 0
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		name := m[2]
		if name == "length" || name == "forEach" {
			continue
		}
		full := prefixes[m[1]] + name
		found++
		if !keys[full] {
			t.Errorf("app.js が読む %q が状態 JSON に無い", full)
		}
	}
	if found < 20 {
		t.Fatalf("app.js から拾えた参照が %d 個しかない(走査パターンが壊れている)", found)
	}
	if !keys["wells.x"] || !keys["obstacles.moving"] || !keys["blackHole.radius"] || !keys["goal.w"] {
		t.Fatal("入れ子のキーが集まっていない(フィクスチャに井戸・障害物・ブラックホールが無い)")
	}
}
