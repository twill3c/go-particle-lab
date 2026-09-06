# Go WebAssembly Particle Game
## Go製リアルタイム物理ゲーム 仕様書

---

# 1. 概要

### 1.1 アプリ名

**Go Particle Lab**

### 1.2 コンセプト

ブラウザ上に大量の粒子を生成し、重力・反発・衝突・風などの物理法則をリアルタイムにシミュレーションする。

物理演算部分をGoで実装し、WebAssembly（Wasm）としてブラウザ上で実行する。

---

# 2. ゲームコンセプト

ユーザーはマウスまたはタッチ操作で「重力場」を作り、落下する粒子を誘導する。

例えば、

```text
                 ●
              ●     ●

          ●           ●

              ╲   ╱
               ╲ ╱
                ◎
          Gravity Well
```

粒子を特定の場所へ集めることがゲームの基本目的。

---

# 3. ゲームモード

## Mode A：Particle Collector

制限時間内に指定数の粒子をGoalへ集める。

```text
Particles
   ↓
   ↓
 ┌─────────┐
 │    ◎    │ Goal
 └─────────┘
```

---

## Mode B：Survival

障害物を避けながら粒子を生存させる。

---

## Mode C：Gravity Puzzle

重力場を配置して粒子をGoalへ導く。

---

# 4. メイン画面

```text
┌───────────────────────────────────────────┐
│              GO PARTICLE LAB              │
├───────────────────────────────────────────┤
│ Score  1,240      Time  42s      ● 500    │
│                                           │
│                                           │
│       ●          ●                        │
│           ●                               │
│                    ●                      │
│                                           │
│    ███████         ●                      │
│                         ◎ GOAL            │
│                                           │
│       ●          ●                        │
│                                           │
├───────────────────────────────────────────┤
│ Gravity     ●──────────────○              │
│ Wind        ○────●──────────              │
│ Particles   ─────●────────                │
│                                           │
│ [ PAUSE ]             [ RESET ]           │
└───────────────────────────────────────────┘
```

---

# 5. Particleモデル

各粒子は以下を持つ。

```go
type Particle struct {
    X        float64
    Y        float64
    VX       float64
    VY       float64
    Radius   float64
    Mass     float64
}
```

---

# 6. Physics Engine

基本物理：

### Gravity

```text
F = m × g
```

### Velocity

```text
velocity += acceleration × deltaTime
```

### Position

```text
position += velocity × deltaTime
```

---

# 7. 壁との衝突

画面端に到達した場合、

```text
VX = -VX
```

または

```text
VY = -VY
```

として跳ね返す。

反発係数を設定する。

```text
Bounce
[────●────────]
```

---

# 8. 粒子同士の衝突

Particle AとParticle Bの距離を計算する。

```text
distance < radiusA + radiusB
```

なら衝突と判断する。

MVPでは完全な物理計算ではなく、簡略化した弾性衝突を採用する。

---

# 9. 重力場

ユーザーが画面をクリックするとGravity Wellを設置する。

```text
             ●
           ↙
        ●
      ↙
    ◎
```

Gravity Well：

```go
type GravityWell struct {
    X         float64
    Y         float64
    Strength  float64
    Radius    float64
}
```

---

# 10. マウス操作

### 左クリック

Gravity Well生成。

### ドラッグ

Gravity Wellを移動。

### 右クリック

Gravity Well削除。

---

# 11. Particle数

設定可能：

```text
Particles

100 ─────────●──────── 5,000
```

推奨MVP：

```text
100～1,000 particles
```

将来的には5,000～10,000粒子を目標とする。

---

# 12. Game Loop

基本構造：

```text
requestAnimationFrame
        ↓
     Update
        ↓
Physics Simulation
        ↓
Collision
        ↓
Rendering
        ↓
requestAnimationFrame
```

Go/Wasm側では物理計算を行い、ブラウザ側では描画を担当する構成を基本とする。

---

# 13. GoとJavaScriptの役割

```text
┌────────────────────────────────────┐
│ Browser                            │
│                                    │
│ Canvas / UI                        │
│       ↕                            │
│ JavaScript                         │
│       ↕                            │
│ WebAssembly                        │
│       ↓                            │
│ Go Physics Engine                  │
└────────────────────────────────────┘
```

### Go

- Physics
- Particle update
- Collision
- Gravity
- Game state
- Score calculation

### JavaScript

- Canvas rendering
- Mouse events
- UI
- WebAssembly bridge

---

# 14. WebAssembly構成

Goコードを、

```text
GOOS=js
GOARCH=wasm
```

でWasmへコンパイルする。

生成：

```text
main.wasm
```

ブラウザからロードする。

---

# 15. データ転送

MVPではJavaScriptからGoへ、

```text
mouseX
mouseY
mouseAction
```

などの入力を渡す。

GoからJavaScriptへ、

```text
particle positions
score
game state
```

などを返す。

大量ParticleではJavaScript ↔ Wasm間のデータ転送がボトルネックになる可能性があるため、将来的にはTypedArrayなどを利用した効率化を検討する。

---

# 16. ゲームルール

### スタート

500個のParticleをランダム配置。

### Goal

画面右側にGoalを配置。

### 目的

制限時間60秒以内に、

```text
Goalへ到達したParticle
```

を100個以上にする。

---

# 17. スコア

基本スコア：

```text
Goal到達
+10

連続到達
Combo Bonus

残り時間
Time Bonus
```

例：

```text
Score
1,240

Combo ×8
```

---

# 18. ステージ

### Stage 1

重力なし。

### Stage 2

通常重力。

### Stage 3

障害物追加。

### Stage 4

複数Gravity Well。

### Stage 5

Wind追加。

### Stage 6

Moving Wall。

### Stage 7

Black Hole。

---

# 19. 障害物

```text
████████████

        ●

───────┐
       │
       │
       └────
```

障害物はParticleの進行方向を変える。

---

# 20. 特殊オブジェクト

将来追加：

### Black Hole

Particleを吸収。

### Repulsor

Particleを反発。

### Portal

別の場所へParticleをワープ。

### Magnet

特定Particleだけを引き寄せる。

---

# 21. Visual Effect

見た目を重視する。

Particleには、

- 軌跡
- 発光
- 衝突エフェクト
- Gravity Wellの波紋
- Goal到達エフェクト

などを追加する。

ただしMVPでは過剰なエフェクトを避け、Physics Engineを優先する。

---

# 22. FPS表示

画面右上：

```text
FPS 60
Particles 1,000
Physics 4.2ms
Render 3.1ms
```

を表示。

これにより、

> 「粒子数を増やすと処理負荷がどう変化するか」

を確認できる。

---

# 23. Physics Benchmark

ゲームとは別にBenchmarkモードを用意する。

```text
Particles    Physics Time
--------------------------
100          0.2 ms
500          1.1 ms
1,000        4.2 ms
2,000        15.3 ms
5,000        91.2 ms
```

※数値は実測値を表示する。

---

# 24. Performance Mode

粒子数を、

```text
100
500
1000
2000
5000
```

と変更してFPSを比較する。

これをゲーム内の「Lab Mode」とする。

---

# 25. ディレクトリ構成

```text
go-particle-lab/
│
├── cmd/
│   └── wasm/
│       └── main.go
│
├── internal/
│   ├── physics/
│   │   ├── particle.go
│   │   ├── gravity.go
│   │   ├── collision.go
│   │   └── world.go
│   │
│   └── game/
│       ├── game.go
│       ├── stage.go
│       └── score.go
│
├── web/
│   ├── index.html
│   ├── app.js
│   ├── style.css
│   └── wasm_exec.js
│
├── tests/
│
├── go.mod
├── Makefile
└── README.md
```

---

# 26. Physics Engineの分離

PhysicsとUIを完全に分離する。

```text
Physics Engine
      │
      ▼
Particle State
      │
      ▼
Game
      │
      ▼
Wasm API
      │
      ▼
Browser
```

これにより、Physics Engine単体のUnit Testが可能になる。

---

# 27. Unit Test

重点テスト：

```text
TestGravity
TestParticleMovement
TestWallCollision
TestParticleCollision
TestGravityWell
TestGoalDetection
TestScore
```

例えば、

```text
Particle
X=100
VX=10

dt=1

期待値
X=110
```

のような決定論的テストを行う。

---

# 28. 開発フェーズ

### Phase 1

1個のParticleを動かす。

### Phase 2

重力。

### Phase 3

複数Particle。

### Phase 4

壁衝突。

### Phase 5

Gravity Well。

### Phase 6

Canvas描画。

### Phase 7

ゲームルール。

### Phase 8

スコア。

### Phase 9

ステージ。

### Phase 10

Benchmark / Lab Mode。

---

# 29. MVP完成条件

- [ ] Go Physics Engine
- [ ] WebAssembly化
- [ ] Canvas表示
- [ ] Particle生成
- [ ] 重力
- [ ] 壁衝突
- [ ] Gravity Well
- [ ] マウス操作
- [ ] Goal
- [ ] スコア
- [ ] 制限時間
- [ ] FPS表示
- [ ] Reset
- [ ] Pause
- [ ] Unit Test
- [ ] Vercel公開
- [ ] GitHub README

---

# 30. 将来拡張

- [ ] 5,000+ particles
- [ ] Particle trails
- [ ] パーティクルカラー変更
- [ ] サウンド
- [ ] ステージエディタ
- [ ] リプレイ
- [ ] ランキング
- [ ] Physics Sandbox
- [ ] Multiplayer
- [ ] WebGPUレンダリング
- [ ] Go Physics Engine Benchmark

---

# 31. 最終コンセプト

このゲームの最大の特徴は、

**Game × Physics × Go × WebAssembly**

である。

単純なWebゲームではなく、

> **「Goで書いた物理エンジンがブラウザ上でリアルタイムに動いていることを体験できるゲーム」**

を目指す。

キャッチコピー：

**Go Particle Lab**

> **Build gravity. Bend particles. Break physics.**