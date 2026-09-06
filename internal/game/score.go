package game

import "math"

// スコア規則(SPEC F-23 / F-24)。
const (
	PointsPerGoal = 10
	ComboWindow   = 1.0 // 秒。直前の到達からこの時間内なら連続到達
	ComboMax      = 8
	TimeBonusRate = 10 // 残り秒(切り捨て)あたり
)

// scoreKeeper はコンボと得点を数える。
type scoreKeeper struct {
	Score      int
	Combo      int
	lastGoalAt float64
	hasGoal    bool
}

// goal は時刻 now(経過秒)に 1 個到達したときの加点を行う。
func (s *scoreKeeper) goal(now float64) {
	if s.hasGoal && now-s.lastGoalAt <= ComboWindow {
		s.Combo++
	} else {
		s.Combo = 1
	}
	s.hasGoal = true
	s.lastGoalAt = now
	mult := s.Combo
	if mult > ComboMax {
		mult = ComboMax
	}
	s.Score += PointsPerGoal * mult
}

// timeBonus は残り秒に対するボーナスを返す(切り捨て × 10)。
func timeBonus(timeLeft float64) int {
	if timeLeft <= 0 {
		return 0
	}
	return int(math.Floor(timeLeft)) * TimeBonusRate
}
