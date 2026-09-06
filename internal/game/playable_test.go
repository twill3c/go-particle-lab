package game

import (
	"math"
	"testing"
)

// carrier は「井戸で粒子を捕まえて Goal へ運ぶ」を機械的に演じる自動プレイヤー(較正用)。
// 周期: gather 秒だけ集める位置に留まる → carry 秒かけて Goal の中心へ直線移動 → hold 秒留まる → 戻る。
type carrier struct {
	gatherFrom, gatherTo [2]float64   // gather 中はこの 2 点の間を往復して掃く
	via                  [][2]float64 // 運搬の経由点(障害物を避ける)。空なら直線
	gather, carry        float64
	hold, back           float64
}

// lerpPath は折れ線 pts を f∈[0,1] で等時間に辿る。
func lerpPath(pts [][2]float64, f float64) (x, y float64) {
	n := len(pts) - 1
	if f <= 0 {
		return pts[0][0], pts[0][1]
	}
	if f >= 1 {
		return pts[n][0], pts[n][1]
	}
	seg := int(f * float64(n))
	if seg >= n {
		seg = n - 1
	}
	u := f*float64(n) - float64(seg)
	a, b := pts[seg], pts[seg+1]
	return a[0] + (b[0]-a[0])*u, a[1] + (b[1]-a[1])*u
}

func (c carrier) wellPos(t float64, goalX, goalY float64) (x, y float64) {
	period := c.gather + c.carry + c.hold + c.back
	u := math.Mod(t, period)
	gx, gy := c.gatherTo[0], c.gatherTo[1]
	switch {
	case u < c.gather:
		// 往復掃引: 0→1→0
		f := 2 * (u / c.gather)
		if f > 1 {
			f = 2 - f
		}
		return c.gatherFrom[0] + (gx-c.gatherFrom[0])*f, c.gatherFrom[1] + (gy-c.gatherFrom[1])*f
	case u < c.gather+c.carry:
		f := (u - c.gather) / c.carry
		pts := append(append([][2]float64{{gx, gy}}, c.via...), [2]float64{goalX, goalY})
		return lerpPath(pts, f)
	case u < c.gather+c.carry+c.hold:
		return goalX, goalY
	default:
		f := (u - c.gather - c.carry - c.hold) / c.back
		return goalX + (gx-goalX)*f, goalY + (gy-goalY)*f
	}
}

// carrierFor はステージに応じた機械的な遊び方。Stage 1 は粒子が上半分にあるので上を掃く。
// 掃引 8 秒 → 運搬 4 秒 → 滞在 2 秒 → 復帰 2 秒(周期 16 秒、60 秒で 3 往復半)。
func carrierFor(stage int) carrier {
	y := WorldHeight - 40
	if stage == 1 {
		y = WorldHeight * 0.25
	}
	return carrier{
		gatherFrom: [2]float64{120, y}, gatherTo: [2]float64{WorldWidth - 200, y},
		gather: 8, carry: 4, hold: 2, back: 2,
	}
}

// stationaryCarrierFor は定点で集める遊び方(掃かない)。集め 6 秒 → 運搬 4 秒 → 滞在 2 秒 → 復帰 2 秒。
func stationaryCarrierFor(stage int) carrier {
	p := [2]float64{WorldWidth * 0.55, WorldHeight - 60}
	switch {
	case stage == 1:
		p = [2]float64{WorldWidth * 0.5, WorldHeight * 0.25}
	case stage >= 5: // 風で左の壁に寄った粒子を取りに行く
		p = [2]float64{WorldWidth * 0.18, WorldHeight - 60}
	}
	c := carrier{gatherFrom: p, gatherTo: p, gather: 6, carry: 4, hold: 2, back: 2}
	if stage >= 3 { // 障害物の下を通ってから上がる
		c.via = [][2]float64{{WorldWidth * 0.85, WorldHeight - 60}}
		c.carry = 6
	}
	return c
}

// playStage は自動プレイで 60 秒(または clear まで)進め、到達数・スコア・状態・残り時間を返す。
func playStage(stage int, seed int64, c carrier) (goaled, score int, status Status, tLeft float64) {
	cfg := DefaultConfig()
	cfg.Stage = stage
	cfg.Seed = seed
	g := New(cfg)
	goalX, goalY := g.Goal.X+g.Goal.W/2, g.Goal.Y+g.Goal.H/2
	x, y := c.wellPos(0, goalX, goalY)
	g.AddWell(x, y)
	for i := 0; i < 60*60 && g.Status == StatusPlaying; i++ {
		x, y = c.wellPos(g.Elapsed, goalX, goalY)
		g.MoveWell(0, x, y)
		g.Update(1.0 / 60)
	}
	return g.Goaled, g.Score, g.Status, g.TimeLeft
}

// 較正ゲート G-07(SPEC §6): 井戸 1 つで「8 秒掃いて集め、4 秒かけて Goal へ運び、2 秒留まる」
// を繰り返す機械的な遊び方で、全 7 ステージが 3 seed とも 60 秒以内にクリアできる。
// 落ちたら井戸の強さ・捕捉半径・減衰(game.go の定数)かステージ配置の問題。
func TestEveryStageClearableByCarrying(t *testing.T) {
	for stage := 1; stage <= 7; stage++ {
		c := stationaryCarrierFor(stage)
		for seed := int64(1); seed <= 3; seed++ {
			goaled, score, status, tl := playStage(stage, seed, c)
			t.Logf("stage %d seed %d: goaled=%d score=%d status=%s timeLeft=%.1f", stage, seed, goaled, score, status, tl)
			if status != StatusClear {
				t.Errorf("stage %d seed %d: 運ぶ自動プレイでクリアできない(goaled=%d)", stage, seed, goaled)
			}
		}
	}
}

// 陰性対照: 井戸を置かなければ Stage 1(重力なし)は 1 個も到達しない。
func TestStage1NeedsAWell(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Stage = 1
	cfg.Seed = 1
	g := New(cfg)
	for i := 0; i < 60*60 && g.Status == StatusPlaying; i++ {
		g.Update(1.0 / 60)
	}
	if g.Goaled != 0 || g.Status != StatusTimeUp {
		t.Fatalf("井戸なしの Stage 1 で goaled=%d status=%s", g.Goaled, g.Status)
	}
}
