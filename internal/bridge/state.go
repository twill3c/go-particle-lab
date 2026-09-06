// Package bridge は Go → JS へ渡す状態の直列化契約を持つ(SPEC F-32 / HC-190)。
// syscall/js に依存しないので、キー名の契約をネイティブの go test で固定できる。
package bridge

import (
	"encoding/json"

	"github.com/twill3c/go-particle-lab/internal/game"
)

// Well は井戸・ブラックホールの描画情報。
type Well struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

// Rect は Goal・障害物の矩形。
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	W      float64 `json:"w"`
	H      float64 `json:"h"`
	Moving bool    `json:"moving"`
}

// State は 1 フレーム分の状態。粒子の位置は含めない(Float32Array で別送)。
type State struct {
	Status    string   `json:"status"`
	Stage     int      `json:"stage"`
	StageName string   `json:"stageName"`
	Score     int      `json:"score"`
	Combo     int      `json:"combo"`
	Goaled    int      `json:"goaled"`
	Target    int      `json:"target"`
	Total     int      `json:"total"`
	Alive     int      `json:"alive"`
	Absorbed  int      `json:"absorbed"`
	TimeLeft  float64  `json:"timeLeft"`
	TimeBonus int      `json:"timeBonus"`
	PhysicsMs float64  `json:"physicsMs"`
	Width     float64  `json:"width"`
	Height    float64  `json:"height"`
	Gravity   float64  `json:"gravity"`
	Wind      float64  `json:"wind"`
	Bounce    float64  `json:"bounce"`
	MaxWells  int      `json:"maxWells"`
	Goal      Rect     `json:"goal"`
	Wells     []Well   `json:"wells"`
	Obstacles []Rect   `json:"obstacles"`
	BlackHole *Well    `json:"blackHole"`
	Stages    []string `json:"stages"`
}

// Build はゲームから State を組む。
func Build(g *game.Game, physicsMs float64) State {
	w := g.World
	s := State{
		Status: string(g.Status), Stage: g.Stage.ID, StageName: g.Stage.Name,
		Score: g.Score, Combo: g.Combo, Goaled: g.Goaled, Target: g.Cfg.Target,
		Total: g.Total, Alive: len(w.Particles), Absorbed: w.Absorbed,
		TimeLeft: g.TimeLeft, TimeBonus: g.TimeBonus, PhysicsMs: physicsMs,
		Width: w.Width, Height: w.Height, Gravity: w.Gravity, Wind: w.Wind, Bounce: w.Bounce,
		MaxWells:  g.Stage.MaxWells,
		Goal:      Rect{X: g.Goal.X, Y: g.Goal.Y, W: g.Goal.W, H: g.Goal.H},
		Wells:     make([]Well, 0, len(w.Wells)),
		Obstacles: make([]Rect, 0, len(w.Obstacles)),
	}
	for _, gw := range w.Wells {
		s.Wells = append(s.Wells, Well{X: gw.X, Y: gw.Y, Radius: gw.Radius})
	}
	for _, o := range w.Obstacles {
		x, y, ow, oh := o.Rect(w.Time)
		s.Obstacles = append(s.Obstacles, Rect{X: x, Y: y, W: ow, H: oh, Moving: o.Amp != 0})
	}
	if bh := w.BlackHole; bh != nil {
		s.BlackHole = &Well{X: bh.X, Y: bh.Y, Radius: bh.EventRadius}
	}
	for _, st := range game.Stages() {
		s.Stages = append(s.Stages, st.Name)
	}
	return s
}

// Marshal は State を JSON 文字列にする。
func Marshal(g *game.Game, physicsMs float64) string {
	b, _ := json.Marshal(Build(g, physicsMs))
	return string(b)
}
