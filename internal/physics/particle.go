// Package physics は粒子の運動則を持つ。UI・syscall/js を一切知らない(SPEC N-02)。
package physics

// Particle は 1 粒子の状態(SPEC F-01)。
type Particle struct {
	X, Y   float64
	VX, VY float64
	Radius float64
	Mass   float64
}
