package game

import (
	"math"
	"testing"
)

// quiet は Stage 1(g=0・風 0・障害物なし)で 500 粒子・seed 固定のゲーム。
// 粒子は初速 0 で力場が無いので、手で動かさない限り止まったまま。
func quiet() *Game {
	cfg := DefaultConfig()
	cfg.Stage = 1
	cfg.Seed = 42
	return New(cfg)
}

// advance は dt クランプ(SPEC §4.1: 1/20 秒)を超えないよう刻んで secs 秒進める。
func advance(g *Game, secs float64) {
	n := int(math.Round(secs / 0.05))
	for i := 0; i < n; i++ {
		g.Update(0.05)
	}
}

// putInGoal は粒子 i を Goal の中心に置く。
func putInGoal(g *Game, i int) {
	g.World.Particles[i].X = g.Goal.X + g.Goal.W/2
	g.World.Particles[i].Y = g.Goal.Y + g.Goal.H/2
}

// T-201 / F-20: N=500・seed 固定で開始 → 粒子 500・全て世界内・初速 0。
func TestInitialPlacement(t *testing.T) {
	g := quiet()
	if n := len(g.World.Particles); n != 500 {
		t.Fatalf("粒子数 %d (期待 500)", n)
	}
	w := g.World
	for i, p := range w.Particles {
		if p.X < p.Radius || p.X > w.Width-p.Radius || p.Y < p.Radius || p.Y > w.Height-p.Radius {
			t.Fatalf("粒子 %d が世界外: (%v,%v)", i, p.X, p.Y)
		}
		if p.VX != 0 || p.VY != 0 {
			t.Fatalf("粒子 %d に初速がある", i)
		}
	}
	if g.Status != StatusPlaying || g.TimeLeft != 60 || g.Score != 0 {
		t.Fatalf("初期状態 status=%s time=%v score=%d", g.Status, g.TimeLeft, g.Score)
	}
}

// T-202 / F-21: Goal 内に粒子を置いて 1 ステップ → 到達 +1・粒子 −1。
func TestGoalDetection(t *testing.T) {
	g := quiet()
	putInGoal(g, 7)
	g.Update(0.05)
	if g.Goaled != 1 || len(g.World.Particles) != 499 {
		t.Fatalf("到達 %d 粒子 %d (期待 1, 499)", g.Goaled, len(g.World.Particles))
	}
	if g.Score != 10 || g.Combo != 1 {
		t.Fatalf("score=%d combo=%d (期待 10, 1)", g.Score, g.Combo)
	}
}

// T-203 / F-23: 0.5 秒間隔で 3 回到達 → 10+20+30、コンボ 3。1.5 秒空けると 1 に戻り 4 回目は +10。
func TestScoreCombo(t *testing.T) {
	g := quiet()
	want := 0
	for k := 0; k < 3; k++ {
		putInGoal(g, 0)
		g.Update(0.05)
		want += 10 * (k + 1)
		if g.Combo != k+1 {
			t.Fatalf("%d 回目: combo=%d", k+1, g.Combo)
		}
		advance(g, 0.45)
	}
	if g.Score != want {
		t.Fatalf("score=%d (期待 %d)", g.Score, want)
	}
	advance(g, 1.5)
	putInGoal(g, 0)
	g.Update(0.05)
	if g.Combo != 1 || g.Score != want+10 {
		t.Fatalf("途切れ後 combo=%d score=%d (期待 1, %d)", g.Combo, g.Score, want+10)
	}
}

// T-204 / F-23: 0.5 秒間隔で 12 回 → 倍率は 8 で頭打ち(9 回目以降 +80)。
func TestComboCap(t *testing.T) {
	g := quiet()
	prev := 0
	for k := 1; k <= 12; k++ {
		putInGoal(g, 0)
		g.Update(0.05)
		gain := g.Score - prev
		prev = g.Score
		wantGain := 10 * k
		if k > 8 {
			wantGain = 80
		}
		if gain != wantGain {
			t.Fatalf("%d 回目の加点 %d (期待 %d)", k, gain, wantGain)
		}
		advance(g, 0.45)
	}
}

// T-205 / F-22 F-24: 目標 100 到達で残り 23.7 秒 → clear・タイムボーナス 230(切り捨て×10)。
func TestClearAndTimeBonus(t *testing.T) {
	g := quiet()
	advance(g, 36.3)
	if math.Abs(g.TimeLeft-23.7) > 1e-6 {
		t.Fatalf("残り %v (期待 ≈23.7)", g.TimeLeft)
	}
	for i := 0; i < 100; i++ {
		putInGoal(g, i)
	}
	scoreBefore := g.Score
	g.Update(0) // dt=0: 積分せず到達判定だけ
	if g.Status != StatusClear || g.Goaled != 100 {
		t.Fatalf("status=%s goaled=%d", g.Status, g.Goaled)
	}
	if g.TimeBonus != 230 {
		t.Fatalf("タイムボーナス %d (期待 230)", g.TimeBonus)
	}
	if g.Score <= scoreBefore+230 {
		t.Fatalf("到達点+ボーナスが加算されていない: %d", g.Score)
	}
	// clear 後は不変。
	score, tl := g.Score, g.TimeLeft
	g.Update(0.05)
	if g.Score != score || g.TimeLeft != tl || g.Status != StatusClear {
		t.Fatalf("clear 後に変化: score %d→%d time %v→%v", score, g.Score, tl, g.TimeLeft)
	}
}

// T-206 / F-22: 60 秒経過 → timeup。以後 step しても状態不変。
func TestTimeUp(t *testing.T) {
	g := quiet()
	g.World.Gravity = 300 // 動く状態で止まることを確かめる
	advance(g, 60)
	if g.Status != StatusTimeUp || g.TimeLeft != 0 {
		t.Fatalf("status=%s time=%v", g.Status, g.TimeLeft)
	}
	before := append([]float64(nil), g.World.Particles[0].X, g.World.Particles[0].Y, float64(g.Score), g.TimeLeft)
	advance(g, 1)
	after := []float64{g.World.Particles[0].X, g.World.Particles[0].Y, float64(g.Score), g.TimeLeft}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("timeup 後に変化: %v → %v", before, after)
		}
	}
}

// T-207 / F-25: 上限 1 の井戸に 2 度 add → 1 個。move/remove は index で効く。
func TestWellLimit(t *testing.T) {
	g := quiet() // Stage 1: 井戸上限 1
	if !g.AddWell(100, 100) {
		t.Fatal("1 個目の追加が拒否された")
	}
	if g.AddWell(200, 200) {
		t.Fatal("上限を超えた追加が受理された")
	}
	if len(g.World.Wells) != 1 {
		t.Fatalf("井戸 %d 個", len(g.World.Wells))
	}
	g.MoveWell(0, 300, 400)
	if w := g.World.Wells[0]; w.X != 300 || w.Y != 400 {
		t.Fatalf("移動後 (%v,%v)", w.X, w.Y)
	}
	g.RemoveWell(0)
	if len(g.World.Wells) != 0 {
		t.Fatal("削除されていない")
	}
	g.RemoveWell(5) // 範囲外は無視
}

// T-208 / F-26: 10 ステップ後 Reset → 初期状態と位置 bit 一致。
func TestReset(t *testing.T) {
	fresh := quiet()
	g := quiet()
	g.AddWell(400, 300)
	advance(g, 0.5)
	g.Reset()
	if len(g.World.Particles) != len(fresh.World.Particles) {
		t.Fatal("粒子数が違う")
	}
	for i := range g.World.Particles {
		if g.World.Particles[i] != fresh.World.Particles[i] {
			t.Fatalf("粒子 %d が不一致", i)
		}
	}
	if len(g.World.Wells) != 0 || g.Score != 0 || g.TimeLeft != 60 || g.Status != StatusPlaying {
		t.Fatalf("Reset 後 wells=%d score=%d time=%v status=%s", len(g.World.Wells), g.Score, g.TimeLeft, g.Status)
	}
}

// T-209 / F-27: ステージ 1〜7 が SPEC §5 の表と一致する。
func TestStages(t *testing.T) {
	type row struct {
		g, wind   float64
		obstacles int
		maxWells  int
		blackHole bool
	}
	want := map[int]row{
		1: {0, 0, 0, 1, false},
		2: {300, 0, 0, 1, false},
		3: {300, 0, 2, 1, false},
		4: {300, 0, 2, 3, false},
		5: {300, -80, 2, 3, false},
		6: {300, -80, 2, 3, false},
		7: {300, -80, 2, 3, true},
	}
	if len(Stages()) != 7 {
		t.Fatalf("ステージ数 %d", len(Stages()))
	}
	for id, r := range want {
		cfg := DefaultConfig()
		cfg.Stage = id
		g := New(cfg)
		w := g.World
		if w.Gravity != r.g || w.Wind != r.wind || len(w.Obstacles) != r.obstacles ||
			g.Stage.MaxWells != r.maxWells || (w.BlackHole != nil) != r.blackHole {
			t.Fatalf("stage %d: g=%v wind=%v obs=%d wells=%d bh=%v", id, w.Gravity, w.Wind, len(w.Obstacles), g.Stage.MaxWells, w.BlackHole != nil)
		}
		if id >= 6 {
			moving := 0
			for _, o := range w.Obstacles {
				if o.Amp != 0 {
					moving++
				}
			}
			if moving != 1 {
				t.Fatalf("stage %d: 移動壁 %d 枚 (期待 1)", id, moving)
			}
		}
	}
}

// T-210 / F-26: Pause 中の Update は時間もスコアも位置も変えない。
func TestPause(t *testing.T) {
	g := quiet()
	g.World.Gravity = 300
	advance(g, 0.5)
	g.Pause()
	x, y, tl := g.World.Particles[0].X, g.World.Particles[0].Y, g.TimeLeft
	advance(g, 1)
	if g.World.Particles[0].X != x || g.World.Particles[0].Y != y || g.TimeLeft != tl || g.Status != StatusPaused {
		t.Fatal("Pause 中に状態が変わった")
	}
	g.Resume()
	advance(g, 0.05)
	if g.World.Particles[0].Y == y {
		t.Fatal("Resume 後に動いていない")
	}
}
