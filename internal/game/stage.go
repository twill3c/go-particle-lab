package game

import "github.com/twill3c/go-particle-lab/internal/physics"

// StageDef はステージの定義(SPEC §5)。
type StageDef struct {
	ID        int
	Name      string
	Gravity   float64
	Wind      float64
	Obstacles []physics.Obstacle
	MaxWells  int
	BlackHole *physics.BlackHole
	UpperHalf bool // 粒子を画面上半分に置く(Stage 1)
}

// 世界の共通寸法(SPEC §5 共通)。
const (
	WorldWidth  = 960.0
	WorldHeight = 600.0
)

var twoRects = []physics.Obstacle{
	{X: 320, Y: 180, W: 160, H: 18},
	{X: 560, Y: 380, W: 160, H: 18},
}

var rectAndMoving = []physics.Obstacle{
	{X: 320, Y: 180, W: 160, H: 18},
	{X: 560, Y: 380, W: 120, H: 18, Amp: 120, Omega: 1.2},
}

// Stages はステージ 1〜7 を返す(SPEC §5 の表)。
func Stages() []StageDef {
	return []StageDef{
		{ID: 1, Name: "Free Float", Gravity: 0, Wind: 0, MaxWells: 1, UpperHalf: true},
		{ID: 2, Name: "Gravity", Gravity: 300, MaxWells: 1},
		{ID: 3, Name: "Obstacles", Gravity: 300, Obstacles: twoRects, MaxWells: 1},
		{ID: 4, Name: "Multi Well", Gravity: 300, Obstacles: twoRects, MaxWells: 3},
		{ID: 5, Name: "Wind", Gravity: 300, Wind: -80, Obstacles: twoRects, MaxWells: 3},
		{ID: 6, Name: "Moving Wall", Gravity: 300, Wind: -80, Obstacles: rectAndMoving, MaxWells: 3},
		{ID: 7, Name: "Black Hole", Gravity: 300, Wind: -80, Obstacles: rectAndMoving, MaxWells: 3,
			BlackHole: &physics.BlackHole{X: WorldWidth / 2, Y: WorldHeight / 2, Strength: 2.5e5, Radius: 220, EventRadius: 14}},
	}
}

// stageByID は ID のステージを返す。範囲外は Stage 1。
func stageByID(id int) StageDef {
	ss := Stages()
	if id < 1 || id > len(ss) {
		return ss[0]
	}
	return ss[id-1]
}
