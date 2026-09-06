package physics

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

const eps = 1e-9

func approx(a, b float64) bool { return math.Abs(a-b) < eps }

func newQuiet(w, h float64) *World {
	wd := NewWorld(w, h)
	wd.Gravity = 0
	wd.Wind = 0
	wd.Bounce = 1
	return wd
}

// T-101 / F-02: 原本 §27 の例。X=100, VX=10, dt=1, 加速度なし → X=110。
func TestParticleMovement(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.AddParticle(Particle{X: 100, Y: 500, VX: 10, Radius: 3, Mass: 1})
	w.Step(1)
	if p := w.Particles[0]; !approx(p.X, 110) || !approx(p.Y, 500) {
		t.Fatalf("X=%v Y=%v (期待 110, 500)", p.X, p.Y)
	}
}

// T-102 / F-02 F-03: g=10, dt=0.5, 初速 0。SPEC §4.1(半陰的オイラー): v' = 5, y' = y0 + v'·dt = y0 + 2.5。
func TestGravity(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.Gravity = 10
	w.AddParticle(Particle{X: 500, Y: 100, Radius: 3, Mass: 1})
	w.Step(0.5)
	p := w.Particles[0]
	if !approx(p.VY, 5) {
		t.Fatalf("VY=%v (期待 5)", p.VY)
	}
	if !approx(p.Y, 102.5) {
		t.Fatalf("Y=%v (期待 102.5 = 100 + 5×0.5)", p.Y)
	}
}

// T-103 / F-04: wind=4, dt=0.25 → VX=1。
func TestWind(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.Wind = 4
	w.AddParticle(Particle{X: 500, Y: 500, Radius: 3, Mass: 1})
	w.Step(0.25)
	if p := w.Particles[0]; !approx(p.VX, 1) {
		t.Fatalf("VX=%v (期待 1)", p.VX)
	}
}

// T-104 / F-05 G-04: 右壁へ VX=100 で突入(e=0.5)→ X=W−r, VX=−50。
// さらにランダム 500 粒子を 200 ステップ進めても全粒子が世界の内側にある。
func TestWallCollision(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.Bounce = 0.5
	w.AddParticle(Particle{X: 996, Y: 500, VX: 100, Radius: 3, Mass: 1})
	w.Step(0.1) // 996 + 10 = 1006 > W − r
	p := w.Particles[0]
	if !approx(p.X, 997) || !approx(p.VX, -50) || !approx(p.Y, 500) {
		t.Fatalf("X=%v VX=%v Y=%v (期待 997, −50, 500)", p.X, p.VX, p.Y)
	}

	w2 := NewWorld(400, 300)
	w2.Bounce = 0.8
	rng := rand.New(rand.NewSource(104))
	for i := 0; i < 500; i++ {
		w2.AddParticle(Particle{
			X: 3 + rng.Float64()*394, Y: 3 + rng.Float64()*294,
			VX: rng.Float64()*800 - 400, VY: rng.Float64()*800 - 400,
			Radius: 3, Mass: 1,
		})
	}
	for s := 0; s < 200; s++ {
		w2.Step(1.0 / 60)
		for i, p := range w2.Particles {
			if p.X < p.Radius || p.X > 400-p.Radius || p.Y < p.Radius || p.Y > 300-p.Radius {
				t.Fatalf("step %d 粒子 %d が外に出た: (%v, %v)", s, i, p.X, p.Y)
			}
		}
	}
}

// T-105 / F-05: e=0 で壁に当たると法線速度 0、接線速度は保存。
func TestWallBounceZero(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.Bounce = 0
	w.AddParticle(Particle{X: 5, Y: 500, VX: -100, VY: 7, Radius: 3, Mass: 1})
	w.Step(0.1)
	p := w.Particles[0]
	if p.VX != 0 || !approx(p.VY, 7) || !approx(p.X, 3) {
		t.Fatalf("VX=%v VY=%v X=%v (期待 0, 7, 3)", p.VX, p.VY, p.X)
	}
}

func momentumEnergy(ps []Particle) (px, py, ke float64) {
	for _, p := range ps {
		px += p.Mass * p.VX
		py += p.Mass * p.VY
		ke += 0.5 * p.Mass * (p.VX*p.VX + p.VY*p.VY)
	}
	return
}

// T-106 / F-06 G-01: e=1 の衝突で運動量ベクトルと運動エネルギーが保存される(力学の保存則)。
// dt=0 の Step は積分せず拘束(衝突・壁)だけを解く。粒子は世界の中央に置き壁に触れない。
func TestParticleCollision(t *testing.T) {
	// 質量 1・1 の正面衝突: 速度が入れ替わる(弾性衝突の解析解)。
	w := newQuiet(1000, 1000)
	w.AddParticle(Particle{X: 500, Y: 500, VX: 10, Radius: 3, Mass: 1})
	w.AddParticle(Particle{X: 505, Y: 500, VX: -10, Radius: 3, Mass: 1})
	w.Step(0)
	a, b := w.Particles[0], w.Particles[1]
	if !approx(a.VX, -10) || !approx(b.VX, 10) || !approx(a.VY, 0) || !approx(b.VY, 0) {
		t.Fatalf("正面衝突後 a=(%v,%v) b=(%v,%v) (期待 −10, 10)", a.VX, a.VY, b.VX, b.VY)
	}
	// めり込み 1 を半分ずつ離す(SPEC §4.2)。
	if !approx(a.X, 499.5) || !approx(b.X, 505.5) {
		t.Fatalf("分離後 aX=%v bX=%v (期待 499.5, 505.5)", a.X, b.X)
	}

	// 質量 1・3 の斜め衝突: 保存則で検算。
	w = newQuiet(1000, 1000)
	w.AddParticle(Particle{X: 500, Y: 500, VX: 30, VY: 4, Radius: 3, Mass: 1})
	w.AddParticle(Particle{X: 504, Y: 503, VX: -5, VY: -2, Radius: 3, Mass: 3})
	px0, py0, ke0 := momentumEnergy(w.Particles)
	w.Step(0)
	px1, py1, ke1 := momentumEnergy(w.Particles)
	if !approx(px0, px1) || !approx(py0, py1) {
		t.Fatalf("運動量 (%v,%v) → (%v,%v)", px0, py0, px1, py1)
	}
	if math.Abs(ke1-ke0)/ke0 > eps {
		t.Fatalf("運動エネルギー %v → %v", ke0, ke1)
	}
	if w.Particles[0].VX == 30 && w.Particles[0].VY == 4 {
		t.Fatal("衝突しているのに速度が変わっていない")
	}

	// 離れつつある対は速度不変。
	w = newQuiet(1000, 1000)
	w.AddParticle(Particle{X: 500, Y: 500, VX: -10, Radius: 3, Mass: 1})
	w.AddParticle(Particle{X: 505, Y: 500, VX: 10, Radius: 3, Mass: 1})
	w.Step(0)
	if w.Particles[0].VX != -10 || w.Particles[1].VX != 10 {
		t.Fatalf("離れつつある対の速度が変わった: %v %v", w.Particles[0].VX, w.Particles[1].VX)
	}
}

func pairKey(p [2]int) int { return p[0]*1_000_000 + p[1] }

func sortPairs(ps [][2]int) []int {
	ks := make([]int, len(ps))
	for i, p := range ps {
		if p[0] > p[1] {
			t := p[0]
			p[0], p[1] = p[1], t
		}
		ks[i] = pairKey(p)
	}
	sort.Ints(ks)
	return ks
}

func equalKeys(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func denseWorld(seed int64) *World {
	w := newQuiet(200, 200)
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < 200; i++ {
		w.AddParticle(Particle{X: 4 + rng.Float64()*192, Y: 4 + rng.Float64()*192, Radius: 4, Mass: 1})
	}
	return w
}

// T-107 / F-07 G-02 G-06: 格子と総当たりの衝突対集合が一致する(二経路一致)。
// 陽性対照: 隣接探索を 4 セルに削った変異体は少なくとも 1 seed で不一致になる。
func TestGridMatchesBruteForce(t *testing.T) {
	mutantCaught := false
	for seed := int64(1); seed <= 20; seed++ {
		w := denseWorld(seed)
		brute := sortPairs(w.CollidingPairsBrute())
		grid := sortPairs(w.CollidingPairsGrid())
		if len(brute) == 0 {
			t.Fatalf("seed %d: 衝突対が 0(フィクスチャが薄すぎる)", seed)
		}
		if !equalKeys(brute, grid) {
			t.Fatalf("seed %d: 総当たり %d 対、格子 %d 対で不一致", seed, len(brute), len(grid))
		}
		mutant := sortPairs(w.collidingPairsGridOffsets(gridOffsets[:4]))
		if !equalKeys(brute, mutant) {
			mutantCaught = true
		}
	}
	if !mutantCaught {
		t.Fatal("G-06: 4 セルの変異体が 20 seed のどれでも総当たりと一致した(テストが衝突の取りこぼしを検出できていない)")
	}
}

func seededWorld(seed int64) *World {
	w := NewWorld(960, 600)
	w.Bounce = 0.6
	w.Wind = -40
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < 300; i++ {
		w.AddParticle(Particle{X: 3 + rng.Float64()*954, Y: 3 + rng.Float64()*594, Radius: 3, Mass: 1})
	}
	w.Wells = append(w.Wells, GravityWell{X: 700, Y: 300, Strength: 2e5, Radius: 400})
	w.Obstacles = append(w.Obstacles, Obstacle{X: 300, Y: 200, W: 200, H: 20, Amp: 100, Omega: 1.5})
	w.BlackHole = &BlackHole{X: 480, Y: 450, Strength: 1e5, Radius: 250, EventRadius: 12}
	return w
}

// T-108 / F-12 G-03: 同 seed・同入力列・600 ステップを二度 → 全位置・全速度が == で一致。
func TestDeterminism(t *testing.T) {
	a, b := seededWorld(108), seededWorld(108)
	for s := 0; s < 600; s++ {
		if s == 200 {
			a.Wells = append(a.Wells, GravityWell{X: 200, Y: 100, Strength: 1e5, Radius: 300})
			b.Wells = append(b.Wells, GravityWell{X: 200, Y: 100, Strength: 1e5, Radius: 300})
		}
		a.Step(1.0 / 60)
		b.Step(1.0 / 60)
	}
	if len(a.Particles) != len(b.Particles) || len(a.Particles) == 300 && a.Absorbed == 0 {
		t.Fatalf("粒子数 %d / %d, 吸収 %d(ブラックホールが効いていないか粒子数が食い違う)", len(a.Particles), len(b.Particles), a.Absorbed)
	}
	for i := range a.Particles {
		if a.Particles[i] != b.Particles[i] {
			t.Fatalf("粒子 %d が不一致: %+v vs %+v", i, a.Particles[i], b.Particles[i])
		}
	}
}

// T-109 / F-08: 井戸 (0,0)・S=1e5・粒子が d=100 に静止、Radius=200、dt=0.01
// → 速度は井戸向きで大きさ S/d²·dt = 0.1。Radius 外(d=300)は不変。d=5 < dMin では有限。
func TestGravityWell(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.Wells = []GravityWell{{X: 0, Y: 0, Strength: 1e5, Radius: 200}}
	w.AddParticle(Particle{X: 100, Y: 0, Radius: 3, Mass: 1})
	w.AddParticle(Particle{X: 300, Y: 0, Radius: 3, Mass: 1})
	w.AddParticle(Particle{X: 5, Y: 0, Radius: 3, Mass: 1})
	w.Step(0.01)
	p := w.Particles[0]
	if !approx(p.VX, -0.1) || !approx(p.VY, 0) {
		t.Fatalf("d=100: v=(%v,%v) (期待 −0.1, 0)", p.VX, p.VY)
	}
	if q := w.Particles[1]; q.VX != 0 || q.VY != 0 {
		t.Fatalf("Radius 外の粒子が動いた: (%v,%v)", q.VX, q.VY)
	}
	r := w.Particles[2]
	want := -1e5 / (WellMinDist * WellMinDist) * 0.01
	if math.IsInf(r.VX, 0) || math.IsNaN(r.VX) || !approx(r.VX, want) {
		t.Fatalf("d=5: VX=%v (期待 %v、dMin で頭打ち)", r.VX, want)
	}
}

// T-110 / F-09: 矩形 [100,200]×[100,200] に左から進入(e=0.5)→ X=100−r、VX×(−e)、VY 不変。
func TestObstacle(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.Bounce = 0.5
	w.Obstacles = []Obstacle{{X: 100, Y: 100, W: 100, H: 100}}
	w.AddParticle(Particle{X: 95, Y: 150, VX: 100, VY: 3, Radius: 3, Mass: 1})
	w.Step(0.1) // 95 + 10 = 105: 左辺からの侵入 8 が最小
	p := w.Particles[0]
	if !approx(p.X, 97) || !approx(p.VX, -50) || !approx(p.VY, 3) {
		t.Fatalf("X=%v VX=%v VY=%v (期待 97, −50, 3)", p.X, p.VX, p.VY)
	}
}

// T-111 / F-10: 移動壁 A=50, ω=1 → t=0 で X0、t=π/2 で X0+50。
func TestMovingWall(t *testing.T) {
	o := Obstacle{X: 300, Y: 200, W: 100, H: 20, Amp: 50, Omega: 1}
	if x, _, _, _ := o.Rect(0); !approx(x, 300) {
		t.Fatalf("t=0: X=%v (期待 300)", x)
	}
	if x, _, _, _ := o.Rect(math.Pi / 2); !approx(x, 350) {
		t.Fatalf("t=π/2: X=%v (期待 350)", x)
	}
}

// T-112 / F-11: EventRadius=10。距離 5 の粒子は 1 ステップで除かれ、距離 100 の粒子は残る。
func TestBlackHole(t *testing.T) {
	w := newQuiet(1000, 1000)
	w.BlackHole = &BlackHole{X: 500, Y: 300, Strength: 0, Radius: 200, EventRadius: 10}
	w.AddParticle(Particle{X: 505, Y: 300, Radius: 3, Mass: 1})
	w.AddParticle(Particle{X: 600, Y: 300, Radius: 3, Mass: 1})
	w.Step(1.0 / 60)
	if len(w.Particles) != 1 || w.Absorbed != 1 {
		t.Fatalf("粒子数 %d 吸収 %d (期待 1, 1)", len(w.Particles), w.Absorbed)
	}
	if !approx(w.Particles[0].X, 600) {
		t.Fatalf("残った粒子が違う: X=%v", w.Particles[0].X)
	}
}
