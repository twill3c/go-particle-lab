// 実ブラウザ検品(TEST_SPEC T-303 / T-304 / T-305)。
// web/ を静的配信し、Chromium で開いて Wasm が動くこと・入力が効くこと・Lab Mode が実測を出すことを確かめる。
// 使い方: node scripts/check-browser.mjs [--url https://...] [--shot out/shot.png]
// Playwright はフリートの gihitsu-kobo/node_modules から借りる(このリポジトリは node 依存を持たない)。
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { extname, join } from "node:path";
import { createRequire } from "node:module";

const require = createRequire("C:/_ClaudeCode/gihitsu-kobo/node_modules/");
const { chromium, devices } = require("playwright");

const args = process.argv.slice(2);
const opt = (k, d) => { const i = args.indexOf(k); return i >= 0 ? args[i + 1] : d; };
const shot = opt("--shot", "out/browser-check.png");
let url = opt("--url", null);

const MIME = { ".html": "text/html; charset=utf-8", ".js": "text/javascript", ".css": "text/css", ".wasm": "application/wasm", ".png": "image/png" };
let server = null;
if (!url) {
  server = createServer(async (req, res) => {
    const p = req.url === "/" ? "/index.html" : req.url.split("?")[0];
    try {
      const body = await readFile(join("web", p));
      res.writeHead(200, { "Content-Type": MIME[extname(p)] || "application/octet-stream" });
      res.end(body);
    } catch {
      res.writeHead(404); res.end("not found");
    }
  });
  await new Promise((r) => server.listen(0, "127.0.0.1", r));
  url = `http://127.0.0.1:${server.address().port}/`;
}

const failures = [];
const check = (cond, msg) => { console.log((cond ? "  ok   " : "  FAIL ") + msg); if (!cond) failures.push(msg); };

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1100, height: 900 } });
const errors = [];
page.on("pageerror", (e) => errors.push(String(e)));
page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });

console.log("open " + url);
await page.goto(url, { waitUntil: "load" });
await page.waitForFunction(() => typeof window.GoParticleLab === "object" && document.getElementById("perf-fps").textContent !== "—", null, { timeout: 20000 });

// T-303: 500 粒子で 1.5 秒描画 → alive 500・FPS 更新・physicsMs > 0
await page.waitForTimeout(1500);
const s1 = await page.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))));
const perf = await page.evaluate(() => ({
  fps: document.getElementById("perf-fps").textContent,
  n: document.getElementById("perf-n").textContent,
  phys: document.getElementById("perf-phys").textContent,
  render: document.getElementById("perf-render").textContent,
  time: document.getElementById("hud-time").textContent,
}));
console.log("T-303", JSON.stringify({ status: s1.status, stage: s1.stage, alive: s1.alive, total: s1.total, timeLeft: s1.timeLeft.toFixed(2), physicsMs: s1.physicsMs }), JSON.stringify(perf));
check(s1.total === 500 && s1.alive <= 500 && s1.alive > 400, "500 粒子で開始し alive が 400 超(重力で Goal に落ちる分は減る)");
check(Number(perf.fps) > 10, "FPS 表示が更新され 10 超 (" + perf.fps + ")");
check(s1.timeLeft < 60 && s1.timeLeft > 55, "時間が減っている (" + s1.timeLeft.toFixed(2) + ")");
check(perf.phys.endsWith("ms") && perf.render.endsWith("ms"), "Physics/Render の ms 表示");

// 位置バッファ: 長さ 2×alive の float32 が世界内に収まる
const buf = await page.evaluate(() => {
  const n = GoParticleLab.count();
  const f = new Float32Array(n * 2);
  GoParticleLab.step(0, new Uint8Array(f.buffer));
  let inside = 0;
  for (let i = 0; i < n; i++) { const x = f[2 * i], y = f[2 * i + 1]; if (x >= 0 && x <= 960 && y >= 0 && y <= 600) inside++; }
  return { n, inside, sample: Array.from(f.slice(0, 4)) };
});
console.log("buffer", JSON.stringify(buf));
check(buf.n > 0 && buf.inside === buf.n, "位置配列 2×n が全て世界内 (" + buf.inside + "/" + buf.n + ")");

// T-304: 左クリックで井戸 1、右クリックで 0
const box = await page.locator("#view").boundingBox();
const cx = box.x + box.width * 0.5, cy = box.y + box.height * 0.5;
await page.mouse.click(cx, cy, { button: "left" });
const wells1 = await page.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))).wells.length);
await page.mouse.move(cx, cy);
await page.mouse.down();
await page.mouse.move(cx + 60, cy + 30, { steps: 5 });
await page.mouse.up();
const moved = await page.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))).wells[0]);
await page.mouse.click(cx + 60, cy + 30, { button: "right" });
const wells0 = await page.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))).wells.length);
console.log("T-304", JSON.stringify({ wells1, moved, wells0 }));
check(wells1 === 1, "左クリックで井戸 1 個");
check(moved && Math.abs(moved.x - 480 - 60 * 960 / box.width) < 3, "ドラッグで井戸が右へ動く (" + (moved && moved.x.toFixed(1)) + ")");
check(wells0 === 0, "右クリックで井戸 0 個");

// 遊べること(G-07 の実ブラウザ版): マウスで井戸を床に置いて 6 秒待ち、4 秒かけて Goal へドラッグして 2 秒留まる。
await page.click("#reset");
// ボタンのクリックでページがスクロールするので canvas の位置を取り直す(2026-09-07 に踏んだ)。
const box2 = await page.locator("#view").boundingBox();
const toPage = (wx, wy) => ({ x: box2.x + wx * box2.width / 960, y: box2.y + wy * box2.height / 600 });
await page.waitForTimeout(3000); // 粒子が床に落ち着くまで
const start = toPage(960 * 0.55, 540), goalPt = toPage(930, 300);
await page.mouse.move(start.x, start.y);
await page.mouse.down();
await page.waitForTimeout(6000);
const STEPS = 120;
for (let i = 1; i <= STEPS; i++) {
  await page.mouse.move(start.x + (goalPt.x - start.x) * i / STEPS, start.y + (goalPt.y - start.y) * i / STEPS);
  await page.waitForTimeout(4000 / STEPS);
}
await page.waitForTimeout(2000);
await page.mouse.up();
const carried = await page.evaluate(() => { const s = JSON.parse(GoParticleLab.step(0, new Uint8Array(0))); return { goaled: s.goaled, score: s.score, combo: s.combo, status: s.status, t: s.timeLeft.toFixed(1), wells: s.wells, alive: s.alive }; });
console.log("carry", JSON.stringify(carried));
check(carried.goaled >= 30, "マウスのドラッグで粒子を Goal へ運べる(到達 " + carried.goaled + ")");
check(carried.score >= carried.goaled * 10, "到達分の得点が入る(score " + carried.score + ")");
await page.mouse.click(goalPt.x, goalPt.y, { button: "right" });

// Pause / Reset / Stage 切替。バナーは状態でなく実際の表示(getComputedStyle)で確かめる —
// .banner{display:flex} が [hidden] に勝って消えなくなる欠陥は状態検査では見えない。
const bannerShown = () => page.evaluate(() => getComputedStyle(document.getElementById("banner")).display !== "none");
await page.click("#reset"); // 運搬検査で clear していると PAUSE が効かないので playing に戻す
await page.waitForTimeout(100);
await page.click("#pause");
const paused = await page.evaluate(() => JSON.parse(GoParticleLab.step(0.016, new Uint8Array(0))).status);
await page.waitForTimeout(100);
const shownWhilePaused = await bannerShown();
await page.click("#pause");
await page.waitForTimeout(100);
const shownAfterResume = await bannerShown();
check(shownWhilePaused, "PAUSE 中はバナーが表示される");
check(!shownAfterResume, "RESUME 後はバナーが消える([hidden] が効く)");
await page.selectOption("#stage", "7");
await page.waitForTimeout(300);
const s7 = await page.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))));
console.log("stage7", JSON.stringify({ paused, stage: s7.stage, obstacles: s7.obstacles.length, bh: !!s7.blackHole, wind: s7.wind, timeLeft: s7.timeLeft.toFixed(2) }));
check(paused === "paused", "PAUSE で paused");
check(s7.stage === 7 && s7.blackHole && s7.obstacles.length === 2 && s7.wind === -80, "Stage 7 に切替(ブラックホール・障害物 2・風 −80)");
check(s7.timeLeft > 59, "ステージ切替で Reset(時間が戻る)");

// T-305: Lab Mode
await page.click("#bench-run");
await page.waitForFunction(() => !document.getElementById("bench-run").disabled, null, { timeout: 60000 });
const rows = await page.$$eval("#bench-body tr", (trs) => trs.map((tr) => Array.from(tr.children).map((td) => td.textContent.trim())));
const statusBeforeShot = await page.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))).status);
console.log("T-305", "status before bench rows:", statusBeforeShot);
for (const r of rows) console.log("   " + r.join(" | "));
check(rows.length === 5 && rows.every((r) => /^\d+(\.\d+)? ms$/.test(r[1])), "5 行すべてに実測 ms が入る");

await page.screenshot({ path: shot, fullPage: true });
console.log("screenshot → " + shot);
check(errors.length === 0, "ページエラー 0 件" + (errors.length ? ": " + errors.join(" / ") : ""));

// 狭い画面(iPhone 13 相当 390px): 横スクロールが出ない・タッチで運べる・バナーがキャンバス内に収まる。
// バナーのはみ出しは数値検査が全部緑のまま起きるので、実際の矩形で見る(2026-09-07)。
const mctx = await browser.newContext({ ...devices["iPhone 13"] });
const mp = await mctx.newPage();
const merr = [];
mp.on("pageerror", (e) => merr.push(String(e)));
await mp.goto(url, { waitUntil: "load" });
await mp.waitForFunction(() => typeof window.GoParticleLab === "object" && document.getElementById("perf-fps").textContent !== "—", null, { timeout: 30000 });
await mp.waitForTimeout(3000);
const doc = await mp.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth }));
const mbox = await mp.locator("#view").boundingBox();
const mp1 = { x: mbox.x + 528 * mbox.width / 960, y: mbox.y + 540 * mbox.height / 600 };
const mp2 = { x: mbox.x + 930 * mbox.width / 960, y: mbox.y + 300 * mbox.height / 600 };
await mp.evaluate(async ([ax, ay, bx, by]) => {
  const c = document.getElementById("view");
  const fire = (t, x, y) => {
    const tp = new Touch({ identifier: 1, target: c, clientX: x, clientY: y });
    c.dispatchEvent(new TouchEvent(t, { touches: t === "touchend" ? [] : [tp], changedTouches: [tp], bubbles: true, cancelable: true }));
  };
  fire("touchstart", ax, ay);
  await new Promise((r) => setTimeout(r, 6000));
  for (let i = 1; i <= 60; i++) { fire("touchmove", ax + (bx - ax) * i / 60, ay + (by - ay) * i / 60); await new Promise((r) => setTimeout(r, 70)); }
  await new Promise((r) => setTimeout(r, 1500));
  fire("touchend", bx, by);
}, [mp1.x, mp1.y, mp2.x, mp2.y]);
await mp.waitForTimeout(300);
const ms = await mp.evaluate(() => JSON.parse(GoParticleLab.step(0, new Uint8Array(0))));
const fit = await mp.evaluate(() => {
  const s = document.getElementById("banner-sub"), c = document.getElementById("view");
  const sr = s.getBoundingClientRect(), cr = c.getBoundingClientRect();
  return { inside: sr.left >= cr.left - 1 && sr.right <= cr.right + 1, clipped: s.scrollWidth > s.clientWidth, text: s.textContent };
});
console.log("mobile", JSON.stringify({ doc, goaled: ms.goaled, status: ms.status, fit }));
check(doc.sw <= doc.cw, "390px で横スクロールが出ない (" + doc.sw + " ≤ " + doc.cw + ")");
check(ms.goaled >= 30, "タッチのドラッグで粒子を運べる(到達 " + ms.goaled + ")");
check(fit.inside && !fit.clipped, "バナーがキャンバス内に収まり、切れていない: " + JSON.stringify(fit.text));
await mp.screenshot({ path: shot.replace(/\.png$/, "-mobile.png"), fullPage: true });
check(merr.length === 0, "モバイルのページエラー 0 件" + (merr.length ? ": " + merr.join(" / ") : ""));

await browser.close();
if (server) server.close();
console.log(failures.length ? `\n${failures.length} 件 FAIL` : "\nALL OK");
process.exit(failures.length ? 1 : 0);
