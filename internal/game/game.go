// Package game はステージ・Goal・スコア・制限時間を持つ。UI・syscall/js を知らない(SPEC N-02)。
package game

import (
	"math/rand"

	"github.com/twill3c/go-particle-lab/internal/physics"
)

// Status はゲームの状態。
type Status string

const (
	StatusPlaying Status = "playing"
	StatusPaused  Status = "paused"
	StatusClear   Status = "clear"
	StatusTimeUp  Status = "timeup"
)

// 既定値(SPEC §5 共通・F-20・F-22)。
const (
	DefaultParticles = 500
	DefaultTimeLimit = 60.0
	DefaultTarget    = 100
	ParticleRadius   = 3.0
	ParticleMass     = 1.0
	MaxDt            = 1.0 / 20 // SPEC §4.1: Update の dt クランプ

	// 井戸の既定値(クリックで置かれる)。SPEC §4.3。
	WellStrength = 1.0e6 // 較正 2026-09-07: 1 往復で運べる粒子が約 80(0.8e6: 50 / 1.3e6: 125)
	WellRadius   = 130.0 // 半径は運搬量にほとんど効かない(90〜160 で同等)
	WellCapture  = 40.0  // この内側で速度が減衰し、粒子が井戸に落ち着く(井戸ごと運べる)
	WellDamping  = 4.0   // 1/s

	GoalWidth  = 60.0
	GoalHeight = 160.0
)

// Config はゲームの生成条件。
type Config struct {
	Width, Height float64
	Particles     int
	Stage         int
	Seed          int64
	TimeLimit     float64
	Target        int
	Gravity       *float64 // nil ならステージの値
	Wind          *float64
	Bounce        *float64
}

// DefaultConfig は SPEC §5 の共通条件。
func DefaultConfig() Config {
	return Config{
		Width: WorldWidth, Height: WorldHeight,
		Particles: DefaultParticles, Stage: 2, Seed: 1,
		TimeLimit: DefaultTimeLimit, Target: DefaultTarget,
	}
}

// Goal は右端の矩形(SPEC F-21)。
type Goal struct {
	X, Y, W, H float64
}

// Game は 1 プレイの状態。
type Game struct {
	Cfg   Config
	Stage StageDef
	World *physics.World
	Goal  Goal

	Score     int
	Combo     int
	Goaled    int
	Total     int
	TimeLeft  float64
	TimeBonus int
	Status    Status
	Elapsed   float64

	keeper scoreKeeper
	paused bool
}

// New は設定からゲームを作る。同じ Config なら同じ初期状態(G-03)。
func New(cfg Config) *Game {
	g := &Game{Cfg: cfg}
	g.Reset()
	return g
}

// Reset は同じステージ・同じ seed で最初からにする(SPEC F-26)。
func (g *Game) Reset() {
	cfg := g.Cfg
	st := stageByID(cfg.Stage)
	g.Stage = st
	w := physics.NewWorld(cfg.Width, cfg.Height)
	w.Gravity = st.Gravity
	w.Wind = st.Wind
	if cfg.Gravity != nil {
		w.Gravity = *cfg.Gravity
	}
	if cfg.Wind != nil {
		w.Wind = *cfg.Wind
	}
	if cfg.Bounce != nil {
		w.Bounce = *cfg.Bounce
	}
	w.Obstacles = append([]physics.Obstacle(nil), st.Obstacles...)
	if st.BlackHole != nil {
		bh := *st.BlackHole
		w.BlackHole = &bh
	}
	g.World = w
	g.Goal = Goal{X: cfg.Width - GoalWidth, Y: cfg.Height/2 - GoalHeight/2, W: GoalWidth, H: GoalHeight}
	g.placeParticles()

	g.Score, g.Combo, g.Goaled, g.TimeBonus = 0, 0, 0, 0
	g.Total = len(w.Particles)
	g.TimeLeft = cfg.TimeLimit
	g.Elapsed = 0
	g.keeper = scoreKeeper{}
	g.paused = false
	g.Status = StatusPlaying
}

// placeParticles は seed 付き乱数で粒子を置く(SPEC F-20)。Goal とブラックホールの近くは避ける。
func (g *Game) placeParticles() {
	rng := rand.New(rand.NewSource(g.Cfg.Seed))
	w := g.World
	r := ParticleRadius
	maxX := g.Goal.X - 20 - r
	maxY := w.Height - r
	if g.Stage.UpperHalf {
		maxY = w.Height / 2
	}
	for len(w.Particles) < g.Cfg.Particles {
		x := r + rng.Float64()*(maxX-r)
		y := r + rng.Float64()*(maxY-r)
		if bh := w.BlackHole; bh != nil {
			dx, dy := x-bh.X, y-bh.Y
			if dx*dx+dy*dy < 4*bh.EventRadius*bh.EventRadius {
				continue
			}
		}
		w.AddParticle(physics.Particle{X: x, Y: y, Radius: r, Mass: ParticleMass})
	}
}

// Update は dt 秒進める。playing 以外では何もしない。dt は [0, MaxDt] に丸める(SPEC §4.1)。
func (g *Game) Update(dt float64) {
	if g.Status != StatusPlaying {
		return
	}
	if dt < 0 {
		dt = 0
	} else if dt > MaxDt {
		dt = MaxDt
	}
	g.World.Step(dt)
	g.Elapsed += dt
	g.TimeLeft = g.Cfg.TimeLimit - g.Elapsed
	// dt を 1,200 回足すと 60 に 1e-12 届かない(浮動小数の累積)。保証粒度は秒なので 1ns 未満は 0 とみなす。
	if g.TimeLeft < 1e-9 {
		g.TimeLeft = 0
	}
	g.collectGoal()
	if g.Goaled >= g.Cfg.Target {
		g.TimeBonus = timeBonus(g.TimeLeft)
		g.Score += g.TimeBonus
		g.Status = StatusClear
		return
	}
	if g.TimeLeft <= 0 {
		g.Status = StatusTimeUp
	}
}

// collectGoal は Goal に入った粒子を世界から除き、加点する(SPEC F-21 / F-23)。順序を保つ。
func (g *Game) collectGoal() {
	w := g.World
	kept := w.Particles[:0]
	for _, p := range w.Particles {
		if p.X >= g.Goal.X && p.X <= g.Goal.X+g.Goal.W && p.Y >= g.Goal.Y && p.Y <= g.Goal.Y+g.Goal.H {
			g.Goaled++
			g.keeper.goal(g.Elapsed)
			continue
		}
		kept = append(kept, p)
	}
	w.Particles = kept
	g.Score = g.keeper.Score + g.TimeBonus
	g.Combo = g.keeper.Combo
}

// AddWell は井戸を置く。上限に達していれば false(SPEC F-25)。
func (g *Game) AddWell(x, y float64) bool {
	if len(g.World.Wells) >= g.Stage.MaxWells {
		return false
	}
	g.World.Wells = append(g.World.Wells, physics.GravityWell{
		X: x, Y: y, Strength: WellStrength, Radius: WellRadius, Capture: WellCapture, Damping: WellDamping,
	})
	return true
}

// MoveWell は井戸 i を動かす。範囲外は無視。
func (g *Game) MoveWell(i int, x, y float64) {
	if i < 0 || i >= len(g.World.Wells) {
		return
	}
	g.World.Wells[i].X = x
	g.World.Wells[i].Y = y
}

// RemoveWell は井戸 i を除く。範囲外は無視。
func (g *Game) RemoveWell(i int) {
	if i < 0 || i >= len(g.World.Wells) {
		return
	}
	g.World.Wells = append(g.World.Wells[:i], g.World.Wells[i+1:]...)
}

// WellAt は点 (x,y) から半径 tol 内で最も近い井戸の添字を返す。無ければ −1。
func (g *Game) WellAt(x, y, tol float64) int {
	best, bestD := -1, tol*tol
	for i, w := range g.World.Wells {
		dx, dy := w.X-x, w.Y-y
		if d := dx*dx + dy*dy; d <= bestD {
			best, bestD = i, d
		}
	}
	return best
}

// Pause は playing を paused にする。
func (g *Game) Pause() {
	if g.Status == StatusPlaying {
		g.Status = StatusPaused
	}
}

// Resume は paused を playing に戻す。
func (g *Game) Resume() {
	if g.Status == StatusPaused {
		g.Status = StatusPlaying
	}
}

// TogglePause は Pause/Resume を切り替える。
func (g *Game) TogglePause() {
	if g.Status == StatusPaused {
		g.Resume()
	} else {
		g.Pause()
	}
}

// SetParam はスライダー値を反映する(SPEC F-28)。particles と stage は Reset を伴う。
// 未知の名前は無視して false を返す。
func (g *Game) SetParam(name string, v float64) bool {
	switch name {
	case "gravity":
		g.Cfg.Gravity = &v
		g.World.Gravity = v
	case "wind":
		g.Cfg.Wind = &v
		g.World.Wind = v
	case "bounce":
		g.Cfg.Bounce = &v
		g.World.Bounce = v
	case "particles":
		g.Cfg.Particles = int(v)
		g.Reset()
	case "stage":
		g.Cfg.Stage = int(v)
		g.Cfg.Gravity, g.Cfg.Wind = nil, nil // ステージの値に戻す
		g.Reset()
	default:
		return false
	}
	return true
}
