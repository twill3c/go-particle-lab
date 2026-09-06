// Go Particle Lab — ブラウザ側。描画・入力・UI だけを担い、粒子の運動則は持たない(SPEC N-03)。
// Go/Wasm 側の契約は cmd/wasm/main.go の先頭コメント。
(function () {
  "use strict";

  const canvas = document.getElementById("view");
  const ctx = canvas.getContext("2d", { alpha: false });
  const W = canvas.width;
  const H = canvas.height;

  const $ = (id) => document.getElementById(id);
  const el = {
    score: $("hud-score"), time: $("hud-time"), goal: $("hud-goal"), combo: $("hud-combo"),
    fps: $("perf-fps"), n: $("perf-n"), phys: $("perf-phys"), render: $("perf-render"),
    banner: $("banner"), bannerBig: $("banner-big"), bannerSub: $("banner-sub"),
    hint: $("hint"), stage: $("stage"),
    particles: $("particles"), gravity: $("gravity"), wind: $("wind"), bounce: $("bounce"),
    particlesVal: $("particles-val"), gravityVal: $("gravity-val"), windVal: $("wind-val"), bounceVal: $("bounce-val"),
    pause: $("pause"), reset: $("reset"), benchBody: $("bench-body"), benchRun: $("bench-run"),
  };

  const STAGE_HINTS = [
    "Stage 1: 重力なし。粒子は動かないので、井戸を置いて捕まえ、ドラッグで Goal(右端の緑)へ運ぶ。",
    "Stage 2: 通常重力。床に落ちた粒子を井戸で捕まえ、ゆっくり持ち上げて Goal へ運ぶ。速く動かすと落とす。",
    "Stage 3: 障害物あり。板を避けて運ぶ。",
    "Stage 4: 井戸を 3 つまで置ける。リレーで運ぶ。",
    "Stage 5: 左向きの風。粒子は左の壁に寄るので、取りに行く。",
    "Stage 6: 動く壁。タイミングを読む。",
    "Stage 7: 中央のブラックホールに落ちた粒子は消える。近づけない。",
  ];

  // Go 側から受け取る位置バッファ。Float32Array(x,y,x,y,…)の buffer を Uint8Array で包んで渡す(F-32)。
  let capacity = 0;
  let f32 = null;
  let u8 = null;
  function ensureBuffer(n) {
    if (n <= capacity) return;
    capacity = Math.max(n, capacity * 2, 1024);
    f32 = new Float32Array(capacity * 2);
    u8 = new Uint8Array(f32.buffer);
  }

  let state = null;
  let lastTs = 0;
  let frames = 0;
  let fpsAt = 0;
  let fps = 0;
  let renderMs = 0;
  let running = false;

  // ---------- 描画 ----------
  function draw(s) {
    const t0 = performance.now();
    ctx.fillStyle = "#05070c";
    ctx.fillRect(0, 0, W, H);

    // 障害物
    for (const o of s.obstacles || []) {
      ctx.fillStyle = o.moving ? "#8c6a3a" : "#3a4a68";
      ctx.fillRect(o.x, o.y, o.w, o.h);
    }

    // Goal
    const g = s.goal;
    ctx.fillStyle = "rgba(74, 222, 128, 0.15)";
    ctx.fillRect(g.x, g.y, g.w, g.h);
    ctx.strokeStyle = "#4ade80";
    ctx.lineWidth = 2;
    ctx.strokeRect(g.x + 1, g.y + 1, g.w - 2, g.h - 2);
    ctx.fillStyle = "#4ade80";
    ctx.font = "bold 12px ui-monospace, Consolas, monospace";
    ctx.textAlign = "center";
    ctx.fillText("GOAL", g.x + g.w / 2, g.y - 6);

    // ブラックホール
    if (s.blackHole) {
      const b = s.blackHole;
      const grad = ctx.createRadialGradient(b.x, b.y, b.radius, b.x, b.y, b.radius * 5);
      grad.addColorStop(0, "rgba(160, 80, 255, 0.35)");
      grad.addColorStop(1, "rgba(160, 80, 255, 0)");
      ctx.fillStyle = grad;
      ctx.beginPath();
      ctx.arc(b.x, b.y, b.radius * 5, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = "#000";
      ctx.beginPath();
      ctx.arc(b.x, b.y, b.radius, 0, Math.PI * 2);
      ctx.fill();
      ctx.strokeStyle = "#b07cff";
      ctx.lineWidth = 1.5;
      ctx.stroke();
    }

    // 井戸
    for (const w of s.wells || []) {
      ctx.strokeStyle = "rgba(90, 209, 255, 0.18)";
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.arc(w.x, w.y, w.radius, 0, Math.PI * 2);
      ctx.stroke();
      const grad = ctx.createRadialGradient(w.x, w.y, 0, w.x, w.y, 40);
      grad.addColorStop(0, "rgba(90, 209, 255, 0.55)");
      grad.addColorStop(1, "rgba(90, 209, 255, 0)");
      ctx.fillStyle = grad;
      ctx.beginPath();
      ctx.arc(w.x, w.y, 40, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = "#5ad1ff";
      ctx.beginPath();
      ctx.arc(w.x, w.y, 6, 0, Math.PI * 2);
      ctx.fill();
    }

    // 粒子(Go が書いた Float32Array を読むだけ)
    const n = s.alive;
    ctx.fillStyle = "#ffd166";
    for (let i = 0; i < n; i++) {
      const x = f32[2 * i];
      const y = f32[2 * i + 1];
      ctx.fillRect(x - 2.5, y - 2.5, 5, 5);
    }

    renderMs = performance.now() - t0;
  }

  // ---------- HUD ----------
  function fmt(n) { return n.toLocaleString("en-US"); }

  function updateHud(s) {
    el.score.textContent = fmt(s.score);
    el.time.innerHTML = s.timeLeft.toFixed(1) + "<small>s</small>";
    el.goal.innerHTML = fmt(s.goaled) + "<small> / " + fmt(s.target) + "</small>";
    el.combo.textContent = "×" + Math.min(Math.max(s.combo, 1), 8) + (s.combo > 8 ? " (" + s.combo + ")" : "");
    el.n.textContent = fmt(s.alive);
    el.phys.textContent = s.physicsMs.toFixed(2) + " ms";
    el.render.textContent = renderMs.toFixed(2) + " ms";
    el.fps.textContent = String(fps);

    if (s.status === "clear") {
      showBanner("clear", "CLEAR", "Score " + fmt(s.score) + "  (time bonus +" + fmt(s.timeBonus) + ")  —  RESET で次へ");
    } else if (s.status === "timeup") {
      showBanner("timeup", "TIME UP", fmt(s.goaled) + " / " + fmt(s.target) + " 到達  —  RESET でもう一度");
    } else if (s.status === "paused") {
      showBanner("paused", "PAUSED", "PAUSE でも Space でも再開");
    } else {
      el.banner.hidden = true;
    }
    el.pause.textContent = s.status === "paused" ? "RESUME" : "PAUSE";
  }

  function showBanner(cls, big, sub) {
    el.banner.className = "banner " + cls;
    el.bannerBig.textContent = big;
    el.bannerSub.textContent = sub;
    el.banner.hidden = false;
  }

  function syncControls(s) {
    el.gravity.value = s.gravity;
    el.gravityVal.textContent = String(Math.round(s.gravity));
    el.wind.value = s.wind;
    el.windVal.textContent = String(Math.round(s.wind));
    el.bounce.value = s.bounce;
    el.bounceVal.textContent = s.bounce.toFixed(2);
    el.stage.value = String(s.stage);
    el.hint.textContent = STAGE_HINTS[s.stage - 1] + "  井戸は " + s.maxWells + " 個まで。";
  }

  // ---------- ループ ----------
  function frame(ts) {
    if (!running) return;
    const dt = lastTs ? (ts - lastTs) / 1000 : 0;
    lastTs = ts;
    ensureBuffer(GoParticleLab.count());
    state = JSON.parse(GoParticleLab.step(dt, u8));
    draw(state);
    updateHud(state);
    frames++;
    if (ts - fpsAt >= 1000) {
      fps = Math.round(frames * 1000 / (ts - fpsAt));
      frames = 0;
      fpsAt = ts;
    }
    requestAnimationFrame(frame);
  }

  function apply(json) {
    state = JSON.parse(json);
    ensureBuffer(state.alive);
    syncControls(state);
    updateHud(state);
  }

  // ---------- 入力(F-43) ----------
  function canvasPoint(ev) {
    const r = canvas.getBoundingClientRect();
    const p = ev.touches ? ev.touches[0] || ev.changedTouches[0] : ev;
    return { x: (p.clientX - r.left) * W / r.width, y: (p.clientY - r.top) * H / r.height };
  }

  let dragIdx = -1;

  function pointerDown(ev) {
    const { x, y } = canvasPoint(ev);
    let idx = GoParticleLab.input("well_at", x, y);
    if (idx < 0) {
      const before = state ? state.wells.length : 0;
      apply(GoParticleLab.input("well_add", x, y));
      idx = state.wells.length > before ? state.wells.length - 1 : -1;
    }
    dragIdx = idx;
  }
  function pointerMove(ev) {
    if (dragIdx < 0) return;
    const { x, y } = canvasPoint(ev);
    GoParticleLab.input("well_move", x, y, dragIdx);
    ev.preventDefault();
  }
  function pointerUp() { dragIdx = -1; }

  canvas.addEventListener("mousedown", (ev) => { if (ev.button === 0) pointerDown(ev); });
  canvas.addEventListener("mousemove", pointerMove);
  window.addEventListener("mouseup", pointerUp);
  canvas.addEventListener("contextmenu", (ev) => {
    ev.preventDefault();
    const { x, y } = canvasPoint(ev);
    const idx = GoParticleLab.input("well_at", x, y);
    if (idx >= 0) apply(GoParticleLab.input("well_remove", 0, 0, idx));
  });
  canvas.addEventListener("touchstart", (ev) => { pointerDown(ev); ev.preventDefault(); }, { passive: false });
  canvas.addEventListener("touchmove", pointerMove, { passive: false });
  canvas.addEventListener("touchend", pointerUp);

  // ---------- コントロール(F-44) ----------
  function bindRange(input, valEl, name, fmtFn) {
    input.addEventListener("input", () => {
      valEl.textContent = fmtFn(Number(input.value));
      apply(GoParticleLab.setParam(name, Number(input.value)));
    });
  }
  bindRange(el.gravity, el.gravityVal, "gravity", (v) => String(v));
  bindRange(el.wind, el.windVal, "wind", (v) => String(v));
  bindRange(el.bounce, el.bounceVal, "bounce", (v) => v.toFixed(2));
  el.particles.addEventListener("input", () => { el.particlesVal.textContent = fmt(Number(el.particles.value)); });
  el.particles.addEventListener("change", () => { apply(GoParticleLab.setParam("particles", Number(el.particles.value))); });
  el.stage.addEventListener("change", () => { apply(GoParticleLab.setParam("stage", Number(el.stage.value))); });
  el.pause.addEventListener("click", () => { apply(GoParticleLab.input("toggle", 0, 0)); });
  el.reset.addEventListener("click", () => { apply(GoParticleLab.input("reset", 0, 0)); });
  window.addEventListener("keydown", (ev) => {
    if (ev.code === "Space" && ev.target === document.body) {
      ev.preventDefault();
      apply(GoParticleLab.input("toggle", 0, 0));
    }
  });

  // ---------- Lab Mode(F-45) ----------
  const BENCH_SIZES = [100, 500, 1000, 2000, 5000];
  const BENCH_STEPS = 120;
  function renderBenchRows(results) {
    el.benchBody.innerHTML = "";
    for (const n of BENCH_SIZES) {
      const tr = document.createElement("tr");
      const ms = results[n];
      const pct = ms == null ? 0 : Math.min(100, ms / 16.667 * 100);
      tr.innerHTML = "<td>" + fmt(n) + "</td>" +
        "<td" + (ms == null ? ' class="pending"' : "") + ">" + (ms == null ? "—" : ms.toFixed(2) + " ms") + "</td>" +
        "<td>" + (ms == null ? "" : pct.toFixed(0) + "%<span class=\"bar\" style=\"width:" + Math.max(2, pct) + "px\"></span>") + "</td>";
      el.benchBody.appendChild(tr);
    }
  }
  renderBenchRows({});
  el.benchRun.addEventListener("click", () => {
    const results = {};
    el.benchRun.disabled = true;
    let i = 0;
    const next = () => {
      if (i >= BENCH_SIZES.length) { el.benchRun.disabled = false; return; }
      const n = BENCH_SIZES[i++];
      results[n] = GoParticleLab.bench(n, BENCH_STEPS);
      renderBenchRows(results);
      setTimeout(next, 30);
    };
    renderBenchRows(results);
    setTimeout(next, 30);
  });

  // ---------- 起動 ----------
  window.onGoParticleLabReady = function () {
    apply(GoParticleLab.init(JSON.stringify({ particles: 500, stage: 2, seed: Date.now() % 1000000 })));
    el.stage.innerHTML = "";
    state.stages.forEach((name, i) => {
      const opt = document.createElement("option");
      opt.value = String(i + 1);
      opt.textContent = "Stage " + (i + 1) + " — " + name;
      el.stage.appendChild(opt);
    });
    el.stage.value = String(state.stage);
    running = true;
    requestAnimationFrame(frame);
  };

  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
    .then((r) => go.run(r.instance))
    .catch((err) => {
      el.hint.textContent = "Wasm の読み込みに失敗: " + err;
      console.error(err);
    });
})();
