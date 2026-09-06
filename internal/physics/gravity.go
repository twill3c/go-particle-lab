package physics

import "math"

// WellMinDist は井戸の逆二乗則の距離下限(SPEC §4.3)。中心付近で発散させない。
const WellMinDist = 20.0

// GravityWell は固定の引力源(SPEC F-08)。Radius の外には効かない。
// Capture 内の粒子は速度に減衰 Damping(1/s)を受けて井戸に落ち着き、井戸ごと運べる(SPEC §4.3)。
type GravityWell struct {
	X, Y     float64
	Strength float64
	Radius   float64
	Capture  float64
	Damping  float64
}

// BlackHole は井戸と同じ引力に加え、EventRadius 内の粒子を吸収する(SPEC F-11)。
type BlackHole struct {
	X, Y        float64
	Strength    float64
	Radius      float64
	EventRadius float64
}

// wellAccel は点 (x,y) が井戸から受ける加速度を返す。Radius 外は 0。
func wellAccel(wx, wy, strength, radius, x, y float64) (ax, ay float64) {
	dx, dy := wx-x, wy-y
	d2 := dx*dx + dy*dy
	if d2 >= radius*radius || d2 == 0 {
		return 0, 0
	}
	d := math.Sqrt(d2)
	dd := d
	if dd < WellMinDist {
		dd = WellMinDist
	}
	a := strength / (dd * dd)
	return a * dx / d, a * dy / d
}

// applyForces は重力・風・井戸・ブラックホールの引力で速度を更新する(SPEC §4.1: v += a·dt)。
func (w *World) applyForces(dt float64) {
	for i := range w.Particles {
		p := &w.Particles[i]
		ax, ay := w.Wind, w.Gravity
		damp := 1.0
		for _, g := range w.Wells {
			gx, gy := wellAccel(g.X, g.Y, g.Strength, g.Radius, p.X, p.Y)
			ax += gx
			ay += gy
			if g.Capture > 0 {
				dx, dy := g.X-p.X, g.Y-p.Y
				if dx*dx+dy*dy < g.Capture*g.Capture {
					f := 1 - g.Damping*dt
					if f < 0 {
						f = 0
					}
					damp *= f
				}
			}
		}
		p.VX *= damp
		p.VY *= damp
		if b := w.BlackHole; b != nil {
			gx, gy := wellAccel(b.X, b.Y, b.Strength, b.Radius, p.X, p.Y)
			ax += gx
			ay += gy
		}
		p.VX += ax * dt
		p.VY += ay * dt
	}
}

// absorb はブラックホールの EventRadius 内の粒子を世界から除く。順序を保つ(決定論 G-03)。
func (w *World) absorb() {
	b := w.BlackHole
	if b == nil {
		return
	}
	r2 := b.EventRadius * b.EventRadius
	kept := w.Particles[:0]
	for _, p := range w.Particles {
		dx, dy := p.X-b.X, p.Y-b.Y
		if dx*dx+dy*dy < r2 {
			w.Absorbed++
			continue
		}
		kept = append(kept, p)
	}
	w.Particles = kept
}
