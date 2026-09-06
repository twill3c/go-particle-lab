package physics

import "math"

// gridOffsets は一様格子で調べる近傍セル(自分 + 隣接 8)。
// 先頭 4 つだけを使う変異体が G-06 の陽性対照になる。
var gridOffsets = [9][2]int{
	{0, 0}, {1, 0}, {0, 1}, {1, 1},
	{-1, 0}, {0, -1}, {-1, -1}, {1, -1}, {-1, 1},
}

// CollidingPairsBrute は総当たり O(n²) で衝突している粒子対 (i<j) を返す(テストのオラクル)。
func (w *World) CollidingPairsBrute() [][2]int {
	var out [][2]int
	ps := w.Particles
	for i := 0; i < len(ps); i++ {
		for j := i + 1; j < len(ps); j++ {
			if overlapping(&ps[i], &ps[j]) {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// CollidingPairsGrid は一様格子で衝突対 (i<j) を返す(SPEC F-07)。
func (w *World) CollidingPairsGrid() [][2]int {
	return w.collidingPairsGridOffsets(gridOffsets[:])
}

func overlapping(a, b *Particle) bool {
	dx, dy := b.X-a.X, b.Y-a.Y
	r := a.Radius + b.Radius
	return dx*dx+dy*dy < r*r
}

// buildGrid は粒子をセルに振り分ける(カウンティングソート・割当てを再利用)。
// セル幅は 2×最大半径(SPEC §4.5)。戻り値はセル数の横幅・縦幅。
func (w *World) buildGrid() (cols, rows int) {
	n := len(w.Particles)
	maxR := 0.0
	for i := range w.Particles {
		if r := w.Particles[i].Radius; r > maxR {
			maxR = r
		}
	}
	w.cell = 2 * maxR
	if w.cell <= 0 {
		w.cell = 1
	}
	cols = int(math.Ceil(w.Width/w.cell)) + 1
	rows = int(math.Ceil(w.Height/w.cell)) + 1
	ncell := cols * rows

	if cap(w.cellOf) < n {
		w.cellOf = make([]int, n)
		w.order = make([]int, n)
	}
	w.cellOf = w.cellOf[:n]
	w.order = w.order[:n]
	if cap(w.cellStart) < ncell+1 {
		w.cellStart = make([]int, ncell+1)
	}
	w.cellStart = w.cellStart[:ncell+1]
	for i := range w.cellStart {
		w.cellStart[i] = 0
	}

	for i := range w.Particles {
		c := w.cellIndex(w.Particles[i].X, w.Particles[i].Y, cols, rows)
		w.cellOf[i] = c
		w.cellStart[c+1]++
	}
	for c := 0; c < ncell; c++ {
		w.cellStart[c+1] += w.cellStart[c]
	}
	// 安定に並べる: 同じセル内では添字昇順(決定論 G-03)。
	fill := w.cellStart
	if cap(w.fill) < ncell {
		w.fill = make([]int, ncell)
	}
	w.fill = w.fill[:ncell]
	copy(w.fill, fill[:ncell])
	for i := range w.Particles {
		c := w.cellOf[i]
		w.order[w.fill[c]] = i
		w.fill[c]++
	}
	return cols, rows
}

func (w *World) cellIndex(x, y float64, cols, rows int) int {
	cx := int(x / w.cell)
	cy := int(y / w.cell)
	if cx < 0 {
		cx = 0
	} else if cx >= cols {
		cx = cols - 1
	}
	if cy < 0 {
		cy = 0
	} else if cy >= rows {
		cy = rows - 1
	}
	return cy*cols + cx
}

// collidingPairsGridOffsets は近傍セルの集合を差し替えられる格子探索。i<j で一度だけ返す。
func (w *World) collidingPairsGridOffsets(offsets [][2]int) [][2]int {
	var out [][2]int
	w.forEachGridPair(offsets, func(i, j int) {
		out = append(out, [2]int{i, j})
	})
	return out
}

// forEachGridPair は格子で見つかる衝突候補のうち実際に重なる対 (i<j) に fn を呼ぶ。
// 走査順は i 昇順・近傍セルの順・セル内添字昇順で固定(決定論 G-03)。
func (w *World) forEachGridPair(offsets [][2]int, fn func(i, j int)) {
	if len(w.Particles) < 2 {
		return
	}
	cols, rows := w.buildGrid()
	ps := w.Particles
	for i := range ps {
		ci := w.cellOf[i]
		cx, cy := ci%cols, ci/cols
		for _, off := range offsets {
			nx, ny := cx+off[0], cy+off[1]
			if nx < 0 || ny < 0 || nx >= cols || ny >= rows {
				continue
			}
			c := ny*cols + nx
			for k := w.cellStart[c]; k < w.cellStart[c+1]; k++ {
				j := w.order[k]
				if j <= i {
					continue
				}
				if overlapping(&ps[i], &ps[j]) {
					fn(i, j)
				}
			}
		}
	}
}

// resolveCollisions は格子で見つけた衝突対に SPEC §4.2 の簡略化した弾性衝突を適用する。
func (w *World) resolveCollisions() {
	e := w.Bounce
	w.forEachGridPair(gridOffsets[:], func(i, j int) {
		a, b := &w.Particles[i], &w.Particles[j]
		dx, dy := b.X-a.X, b.Y-a.Y
		d := math.Sqrt(dx*dx + dy*dy)
		var nx, ny float64
		if d > 0 {
			nx, ny = dx/d, dy/d
		} else {
			nx, ny = 1, 0 // 完全に重なった対は x 方向に離す
		}
		// めり込みを半分ずつ押し戻す。
		pen := a.Radius + b.Radius - d
		a.X -= nx * pen / 2
		a.Y -= ny * pen / 2
		b.X += nx * pen / 2
		b.Y += ny * pen / 2
		// 近づいている対だけ速度を交換する。
		vn := (a.VX-b.VX)*nx + (a.VY-b.VY)*ny
		if vn <= 0 {
			return
		}
		jimp := (1 + e) * vn / (1/a.Mass + 1/b.Mass)
		a.VX -= jimp / a.Mass * nx
		a.VY -= jimp / a.Mass * ny
		b.VX += jimp / b.Mass * nx
		b.VY += jimp / b.Mass * ny
	})
}

// Obstacle は軸平行矩形(SPEC F-09)。Amp≠0 なら X = X0 + Amp·sin(Omega·t) で往復する(F-10)。
type Obstacle struct {
	X, Y, W, H float64
	Amp, Omega float64
}

// Rect は時刻 t における矩形を返す。
func (o Obstacle) Rect(t float64) (x, y, w, h float64) {
	x = o.X
	if o.Amp != 0 {
		x += o.Amp * math.Sin(o.Omega*t)
	}
	return x, o.Y, o.W, o.H
}

// resolveObstacles は矩形に重なった粒子を最小侵入軸で押し戻し、その軸の速度を −e 倍にする。
func (w *World) resolveObstacles() {
	e := w.Bounce
	for _, o := range w.Obstacles {
		ox, oy, ow, oh := o.Rect(w.Time)
		for i := range w.Particles {
			p := &w.Particles[i]
			r := p.Radius
			if p.X+r <= ox || p.X-r >= ox+ow || p.Y+r <= oy || p.Y-r >= oy+oh {
				continue
			}
			left := p.X + r - ox
			right := ox + ow - (p.X - r)
			top := p.Y + r - oy
			bottom := oy + oh - (p.Y - r)
			m := left
			axis := 0
			if right < m {
				m, axis = right, 1
			}
			if top < m {
				m, axis = top, 2
			}
			if bottom < m {
				m, axis = bottom, 3
			}
			switch axis {
			case 0:
				p.X = ox - r
				if p.VX > 0 {
					p.VX = -e * p.VX
				}
			case 1:
				p.X = ox + ow + r
				if p.VX < 0 {
					p.VX = -e * p.VX
				}
			case 2:
				p.Y = oy - r
				if p.VY > 0 {
					p.VY = -e * p.VY
				}
			case 3:
				p.Y = oy + oh + r
				if p.VY < 0 {
					p.VY = -e * p.VY
				}
			}
		}
	}
}

// resolveWalls は世界の境界からはみ出た粒子を押し戻し、法線速度を −e 倍にする(SPEC F-05)。
func (w *World) resolveWalls() {
	e := w.Bounce
	for i := range w.Particles {
		p := &w.Particles[i]
		r := p.Radius
		if p.X < r {
			p.X = r
			if p.VX < 0 {
				p.VX = -e * p.VX
			}
		} else if p.X > w.Width-r {
			p.X = w.Width - r
			if p.VX > 0 {
				p.VX = -e * p.VX
			}
		}
		if p.Y < r {
			p.Y = r
			if p.VY < 0 {
				p.VY = -e * p.VY
			}
		} else if p.Y > w.Height-r {
			p.Y = w.Height - r
			if p.VY > 0 {
				p.VY = -e * p.VY
			}
		}
	}
}
