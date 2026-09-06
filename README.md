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
| L2 | ゲーム(配置・Goal・スコア・コンボ・制限時間・ステージ 1〜7・Pause/Reset) | 完了(2026-09-06)・テスト 10 件 |
| L3 | Wasm ブリッジ・Canvas・HUD・FPS・Lab Mode・実ブラウザ検品・遊び方の較正 | 完了(2026-09-07) |
| L4 | GitHub・Vercel・文書 | 未着手 |

## 遊び方

左クリックで重力井戸(Gravity Well)を置く。井戸の内側に入った粒子は速度が減衰して井戸に落ち着くので、
**井戸をドラッグすると粒子ごと運べる**。床に落ちた粒子の上に井戸を置いて数秒待ち、ゆっくり右端の
Goal まで運ぶ。速く動かすと粒子を落とす。右クリックで井戸を消す。60 秒で 100 個集めればクリア。
連続到達でコンボ(最大 ×8)、クリア時は残り秒 × 10 のタイムボーナス。

| Stage | 何が変わるか |
|---|---|
| 1 Free Float | 重力なし。粒子は動かないので井戸で捕まえに行く |
| 2 Gravity | 通常重力 |
| 3 Obstacles | 板 2 枚。避けて運ぶ |
| 4 Multi Well | 井戸を 3 つまで置ける |
| 5 Wind | 左向きの風。粒子は左壁に寄る |
| 6 Moving Wall | 板の 1 枚が往復する |
| 7 Black Hole | 中央のブラックホールが粒子を吸い込む |

Lab Mode(画面下)は 100〜5,000 粒子で Go/Wasm の物理 1 ステップの時間を実測する。
このブラウザでの実測(2026-09-07・Chromium headless・Windows): 1,000 粒子 0.25 ms、5,000 粒子 2.2 ms。

## 動かす

```bash
export PATH="$USERPROFILE/sdk/go/bin:$PATH"   # Go 1.27.1
make test        # go test ./...
make wasm        # web/main.wasm を生成
make serve       # http://localhost:8080
node scripts/check-browser.mjs   # 実ブラウザ検品(Playwright はフリートの gihitsu-kobo/node_modules を借りる)
```

## 構成

```
cmd/wasm/        syscall/js ブリッジ(位置は Float32Array へバイトコピー)
internal/physics 粒子・重力・壁・衝突(一様格子)・井戸(捕捉減衰)・障害物・移動壁・ブラックホール
internal/game    ステージ・Goal・スコア・制限時間・自動プレイの較正ゲート
internal/bridge  Go → JS の状態 JSON の契約(キー名をテストで固定)
web/             index.html / app.js / style.css / wasm_exec.js / main.wasm
tests/           足場の不変量
```

## ライセンス

MIT License © 2026 坂田哲朗
