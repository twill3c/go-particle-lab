package physics

// World は粒子・力場・障害物を持ち、Step(dt) で決定論的に状態を進める(SPEC F-12)。
type World struct {
	Width, Height float64

	Gravity float64 // 一様重力の加速度(下向き正)。F-03
	Wind    float64 // 風の加速度(右向き正)。F-04
	Bounce  float64 // 反発係数 e ∈ [0,1]。壁・障害物・粒子衝突に共通

	Particles []Particle
	Wells     []GravityWell
	Obstacles []Obstacle
	BlackHole *BlackHole

	Time     float64 // 経過秒(移動壁の位相に使う)
	Absorbed int     // ブラックホールに吸収された粒子数

	// 一様格子の作業領域(割当てを再利用する)。
	cell      float64
	cellOf    []int
	order     []int
	cellStart []int
	fill      []int
}

// NewWorld は幅 w・高さ h の世界を返す。既定は g=300, wind=0, e=0.6(SPEC §5 共通)。
func NewWorld(w, h float64) *World {
	return &World{Width: w, Height: h, Gravity: 300, Wind: 0, Bounce: 0.6}
}

// AddParticle は粒子を追加し、その添字を返す。
func (w *World) AddParticle(p Particle) int {
	w.Particles = append(w.Particles, p)
	return len(w.Particles) - 1
}

// Remove は添字 i の粒子を除く。順序を保つ(決定論 G-03)。
func (w *World) Remove(i int) {
	w.Particles = append(w.Particles[:i], w.Particles[i+1:]...)
}

// Step は dt 秒だけ世界を進める。順序: 力 → 積分 → 粒子衝突 → 障害物 → 壁 → 吸収。
// dt=0 は積分せず拘束だけを解く。dt のクランプは呼び出し側(game)の責務。
func (w *World) Step(dt float64) {
	w.applyForces(dt)
	for i := range w.Particles {
		p := &w.Particles[i]
		p.X += p.VX * dt
		p.Y += p.VY * dt
	}
	w.resolveCollisions()
	w.resolveObstacles()
	w.resolveWalls()
	w.absorb()
	w.Time += dt
}
