package physics

import (
	"math/rand"
	"testing"
)

// benchWorld は SPEC §5 の共通条件(960×600・r=3・e=0.6・g=300)で n 粒子の世界を作る。
func benchWorld(n int, seed int64) *World {
	w := NewWorld(960, 600)
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < n; i++ {
		w.AddParticle(Particle{X: 3 + rng.Float64()*954, Y: 3 + rng.Float64()*594, Radius: 3, Mass: 1})
	}
	w.Wells = append(w.Wells, GravityWell{X: 700, Y: 300, Strength: 2e5, Radius: 400})
	return w
}

func benchStep(b *testing.B, n int) {
	w := benchWorld(n, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Step(1.0 / 60)
	}
}

func BenchmarkStep100(b *testing.B)  { benchStep(b, 100) }
func BenchmarkStep500(b *testing.B)  { benchStep(b, 500) }
func BenchmarkStep1000(b *testing.B) { benchStep(b, 1000) }
func BenchmarkStep2000(b *testing.B) { benchStep(b, 2000) }
func BenchmarkStep5000(b *testing.B) { benchStep(b, 5000) }
