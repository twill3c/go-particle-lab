//go:build js && wasm

// cmd/wasm は internal/game を syscall/js でブラウザに橋渡しする(SPEC §2.3)。
// 物理・ゲーム規則はここに書かない。JS 側との契約:
//
//	GoParticleLab.init(jsonConfig)            → state JSON
//	GoParticleLab.step(dt, u8)                → state JSON。u8 は Float32Array の buffer を包む Uint8Array。
//	                                              先頭 2×n 個の float32 に (x,y) を書く(F-32)
//	GoParticleLab.input(action, x, y, idx)    → state JSON。action: well_add / well_move / well_remove /
//	                                              well_at(戻りは idx) / pause / resume / toggle / reset
//	GoParticleLab.setParam(name, value)       → state JSON
//	GoParticleLab.bench(n, steps)             → 1 ステップあたりの ms(実測)
//	GoParticleLab.count()                     → 粒子数
package main

import (
	"encoding/json"
	"math"
	"syscall/js"
	"time"
	"unsafe"

	"github.com/twill3c/go-particle-lab/internal/bridge"
	"github.com/twill3c/go-particle-lab/internal/game"
)

var (
	g         *game.Game
	positions []float32
	physicsMs float64
)

// state は状態 JSON を返す。キー名の契約は internal/bridge が持ち T-306 で固定する(HC-190)。
func state() string {
	return bridge.Marshal(g, physicsMs)
}

type initJSON struct {
	Particles int   `json:"particles"`
	Stage     int   `json:"stage"`
	Seed      int64 `json:"seed"`
}

func jsInit(_ js.Value, args []js.Value) any {
	cfg := game.DefaultConfig()
	if len(args) > 0 && args[0].Type() == js.TypeString {
		var in initJSON
		if err := json.Unmarshal([]byte(args[0].String()), &in); err == nil {
			if in.Particles > 0 {
				cfg.Particles = in.Particles
			}
			if in.Stage > 0 {
				cfg.Stage = in.Stage
			}
			if in.Seed != 0 {
				cfg.Seed = in.Seed
			}
		}
	}
	g = game.New(cfg)
	return state()
}

// jsStep は dt 秒進め、位置を u8(Uint8Array)へ書き、状態 JSON を返す。
func jsStep(_ js.Value, args []js.Value) any {
	dt := args[0].Float()
	t0 := time.Now()
	g.Update(dt)
	physicsMs = float64(time.Since(t0).Nanoseconds()) / 1e6
	if len(args) > 1 {
		fillPositions(args[1])
	}
	return state()
}

// fillPositions は粒子の (x,y) を float32 で詰め、u8 にバイトコピーする(SPEC §4.4)。
func fillPositions(u8 js.Value) {
	ps := g.World.Particles
	n := len(ps)
	if cap(positions) < 2*n {
		positions = make([]float32, 2*n)
	}
	positions = positions[:2*n]
	for i, p := range ps {
		positions[2*i] = float32(p.X)
		positions[2*i+1] = float32(p.Y)
	}
	if n == 0 {
		return
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&positions[0])), 8*n)
	if u8.Length() < len(bytes) {
		bytes = bytes[:u8.Length()]
	}
	js.CopyBytesToJS(u8, bytes)
}

func jsInput(_ js.Value, args []js.Value) any {
	action := args[0].String()
	var x, y float64
	idx := -1
	if len(args) > 2 {
		x, y = args[1].Float(), args[2].Float()
	}
	if len(args) > 3 && args[3].Type() == js.TypeNumber {
		idx = args[3].Int()
	}
	switch action {
	case "well_add":
		g.AddWell(x, y)
	case "well_move":
		g.MoveWell(idx, x, y)
	case "well_remove":
		g.RemoveWell(idx)
	case "well_at":
		return g.WellAt(x, y, 28)
	case "pause":
		g.Pause()
	case "resume":
		g.Resume()
	case "toggle":
		g.TogglePause()
	case "reset":
		g.Reset()
	}
	return state()
}

func jsSetParam(_ js.Value, args []js.Value) any {
	g.SetParam(args[0].String(), args[1].Float())
	return state()
}

// jsBench は n 粒子の世界を steps 回進め、1 ステップあたりの ms を返す(SPEC F-45・実測値)。
func jsBench(_ js.Value, args []js.Value) any {
	n, steps := args[0].Int(), args[1].Int()
	cfg := game.DefaultConfig()
	cfg.Particles = n
	cfg.Stage = 4
	cfg.Seed = 7
	bg := game.New(cfg)
	bg.AddWell(700, 300)
	bg.AddWell(300, 200)
	// ゲーム規則(Goal 到達で粒子が減る)がベンチを歪めないよう、物理だけを回す。
	w := bg.World
	t0 := time.Now()
	for i := 0; i < steps; i++ {
		w.Step(1.0 / 60)
	}
	ms := float64(time.Since(t0).Nanoseconds()) / 1e6 / float64(steps)
	return math.Round(ms*1000) / 1000
}

func jsCount(_ js.Value, _ []js.Value) any {
	return len(g.World.Particles)
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("init", js.FuncOf(jsInit))
	api.Set("step", js.FuncOf(jsStep))
	api.Set("input", js.FuncOf(jsInput))
	api.Set("setParam", js.FuncOf(jsSetParam))
	api.Set("bench", js.FuncOf(jsBench))
	api.Set("count", js.FuncOf(jsCount))
	js.Global().Set("GoParticleLab", api)
	if ready := js.Global().Get("onGoParticleLabReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}
	select {}
}
