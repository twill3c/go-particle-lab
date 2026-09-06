# CLAUDE.md

@AGENTS.md

上記ハーネスがこのリポジトリの正本ルール。要点のみ再掲する:

- 仕様の正本は SPEC.md。変更は スペック → テスト → 実装 の順。
- すべてのタスクは 7 段階ループプロトコル(AGENTS.md の共通規律)で進め、
  `python harness/looplog.py append` で `logs/loops/{loop_id}.jsonl` に記録する。
  失敗は気づいた瞬間に FAILURE_TAXONOMY のコード付きで記録する。
- 完了条件は `go test ./...` green + `make wasm` green + `looplog.py validate` 合格。
- 物理は全て Go(internal/physics)。`web/app.js` は描画とイベントのグルーのみで運動則を知らない(N-03)。
  `internal/` は `syscall/js` を import しない(N-02)。
- 決定性(同一 seed + 同一入力列 → 同一状態、G-03)を壊さない。
- scaffold ブロック(AGENTS.md 冒頭)と `.wt/gate.json` の上限は直接編集しない。
- Go は `~/sdk/go`(1.27.1)にあり PATH 未登録。`export PATH="$USERPROFILE/sdk/go/bin:$PATH"` を通す。
