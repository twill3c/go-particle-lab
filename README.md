# Go Particle Lab

**Build gravity. Bend particles. Break physics.**

ブラウザ上に数百〜数千の粒子を生成し、重力・壁との反発・粒子同士の衝突・風・重力井戸を
リアルタイムに計算するゲーム。**物理演算はすべて Go で書き、WebAssembly としてブラウザで走る。**
JavaScript は描画と入力だけを担い、粒子の運動則を一切知らない。

クリックで重力井戸(Gravity Well)を置き、粒子を画面右の Goal へ導く。60 秒で 100 個集めればクリア。
粒子数を 100〜5,000 に変えて物理時間の変化を見る Lab Mode を併設する。

構想の原本は [docs/go-particle-lab-spec.md](docs/go-particle-lab-spec.md)、
要求 ID 付きの正本は [SPEC.md](SPEC.md)、テストの対応表は [TEST_SPEC.md](TEST_SPEC.md)。

## 状態

| ループ | 範囲 | 状態 |
|---|---|---|
| L0 | 足場・SPEC/TEST_SPEC・go.mod・Makefile | 完了(2026-09-06) |
| L1 | 物理エンジン(積分・重力・風・壁・粒子衝突・一様格子・井戸・障害物・移動壁・ブラックホール) | 完了(2026-09-06)・テスト 12 件 |
| L2 | ゲーム(配置・Goal・スコア・コンボ・制限時間・ステージ 1〜7・Pause/Reset) | 未着手 |
| L3 | Wasm ブリッジ・Canvas・HUD・FPS・Lab Mode | 未着手 |
| L4 | GitHub・Vercel・文書 | 未着手 |

## 動かす

```bash
export PATH="$USERPROFILE/sdk/go/bin:$PATH"   # Go 1.27.1
make test        # go test ./...
make wasm        # web/main.wasm を生成
make serve       # http://localhost:8080
```

## 構成

```
cmd/wasm/        syscall/js ブリッジ(位置は Float32Array へバイトコピー)
internal/physics 粒子・重力・壁・衝突(一様格子)・井戸・障害物
internal/game    ステージ・Goal・スコア・制限時間
web/             index.html / app.js / style.css / wasm_exec.js / main.wasm
tests/           足場の不変量
```

## ライセンス

MIT License © 2026 坂田哲朗
