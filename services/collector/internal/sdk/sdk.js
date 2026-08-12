/*
 * Ghost Trace browser SDK.
 *
 * Collects interaction dynamics: pointer geometry, key timing with a
 * coarse class (never the key), scroll, focus, visibility and form
 * events, plus a once-per-session client block of normalization
 * properties. Every channel and field is enumerated in
 * internal/ingest/vocabulary.go and disclosed in
 * experiments/PARTICIPANTS.md — tests hold all three to each other.
 *
 * No canvas, WebGL, font enumeration or audio fingerprinting: those
 * identify the browser rather than the behaviour and are out of scope
 * per contract/architecture.md §0.
 *
 * Nothing persistent is written to the client. The session token lives
 * in a closure and dies with the page.
 */
(function () {
  "use strict";

  var API = window.GHOST_TRACE_API || "";
  var SITE_KEY = window.GHOST_TRACE_SITE_KEY;

  var token = null;
  var collect = { pointer_hz: 20, batch_ms: 2000, types: ["pointer"] };
  var startedAt = performance.now();

  // Pending polyline and the batch queue.
  var pts = [];
  var queue = [];
  var seq = 0;
  var lastSampleAt = 0;
  var polylineStartedAt = 0;

  var listeners = [];

  function now() {
    return Math.round(performance.now() - startedAt);
  }

  function emit(name, detail) {
    listeners.forEach(function (fn) {
      try { fn(name, detail); } catch (e) { /* a listener must not break collection */ }
    });
  }

  /* --------------------------------------------------------------
   * Session handshake
   * ------------------------------------------------------------ */
  function start() {
    // Normalization properties only. The admission test per §0: does
    // this change how behaviour should be interpreted? Trackpad and
    // mouse produce structurally different traces; a 400px viewport
    // scrolls differently than a 1440px one.
    var client = {
      pointer: window.matchMedia("(pointer: fine)").matches ? "fine"
             : window.matchMedia("(pointer: coarse)").matches ? "coarse"
             : "none",
      touch: ("ontouchstart" in window) || navigator.maxTouchPoints > 0,
      viewport: [window.innerWidth, window.innerHeight],
      tz_offset: -new Date().getTimezoneOffset(),
      reduced_motion: window.matchMedia("(prefers-reduced-motion: reduce)").matches
    };

    return fetch(API + "/v1/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        site_key: SITE_KEY,
        page: { path: location.pathname },
        client: client
      })
    })
      .then(function (r) { return r.json(); })
      .then(function (body) {
        token = body.session_token;
        if (body.collect) {
          // Server-driven. SDKs must tolerate unknown keys and
          // unrequested types (§7), so merge rather than replace.
          if (body.collect.pointer_hz) collect.pointer_hz = body.collect.pointer_hz;
          if (body.collect.batch_ms) collect.batch_ms = body.collect.batch_ms;
          if (body.collect.types) collect.types = body.collect.types;
        }
        attach();
        setInterval(flush, collect.batch_ms);
        emit("session", { token: token, collect: collect });
        return token;
      });
  }

  /* --------------------------------------------------------------
   * Pointer capture
   * ------------------------------------------------------------ */
  function attach() {

    attachKeys();
    attachScroll();
    attachFocus();
    attachVisibility();
    attachForm();

    if (collect.types.indexOf("pointer") === -1) return;
    window.addEventListener("pointermove", function (e) {
      var t = now();
      // Fixed-rate decimation. Raw pointermove is several hundred
      // events per second of movement; M1 ships the simplest possible
      // scheme and §8.2 measures what it destroys.
      var minGap = 1000 / collect.pointer_hz;
      if (t - lastSampleAt < minGap) return;

      var dt = pts.length === 0 ? 0 : t - lastSampleAt;
      if (pts.length === 0) polylineStartedAt = t;
      lastSampleAt = t;

      pts.push([Math.round(e.clientX), Math.round(e.clientY), dt]);
      if (pts.length >= 256) cutPolyline(srcOf(e));
    }, { passive: true });
  }

  /* --------------------------------------------------------------
   * Key capture — TIMING AND COARSE CLASS ONLY
   * ------------------------------------------------------------ */
  // Key CONTENT is never collected. `class` preserves cadence and
  // digraph structure while making reconstruction of typed text
  // infeasible, and `target` is a stable hash of the field identity,
  // never its value (§2, §6).
  function classOf(e) {
    var k = e.key;
    if (!k) return "other";
    if (k.length === 1) {
      if (/[a-zA-Z]/.test(k)) return "alpha";
      if (/[0-9]/.test(k)) return "digit";
      if (k === " ") return "whitespace";
      return "other";
    }
    if (k === "Tab" || k === "Enter" || k.indexOf("Arrow") === 0 ||
        k === "Home" || k === "End" || k === "PageUp" || k === "PageDown" ||
        k === "Backspace" || k === "Delete") return "nav";
    if (k === "Shift" || k === "Control" || k === "Alt" || k === "Meta" ||
        k === "CapsLock") return "mod";
    return "other";
  }

  // Field identity hash. FNV-1a over a structural descriptor — never the
  // field's value, which never leaves the page.
  var targetCache = new WeakMap();
  function targetOf(el) {
    if (!el || el === document || el === window) return "";
    if (targetCache.has(el)) return targetCache.get(el);
    var desc = (el.tagName || "") + "|" + (el.type || "") + "|" +
               (el.name || "") + "|" + (el.id || "");
    var h = 0x811c9dc5;
    for (var i = 0; i < desc.length; i++) {
      h ^= desc.charCodeAt(i);
      h = (h + ((h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24))) >>> 0;
    }
    var id = "f_" + h.toString(16);
    targetCache.set(el, id);
    return id;
  }

  // Last key / paste time per field, for injection detection.
  var lastKeyAt = {};
  var lastPasteAt = {};

  function attachKeys() {
    if (collect.types.indexOf("key") === -1) return;
    ["keydown", "keyup"].forEach(function (type) {
      window.addEventListener(type, function (e) {
        var id = targetOf(e.target);
        var t = now();
        lastKeyAt[id] = t;
        queue.push({
          type: "key", t: t,
          phase: type === "keydown" ? "down" : "up",
          class: classOf(e),
          target: id
        });
      }, { passive: true, capture: true });
    });
  }

  function attachScroll() {
    if (collect.types.indexOf("scroll") === -1) return;
    var lastY = window.scrollY;
    var sawUserGesture = false;
    ["wheel", "touchmove", "keydown"].forEach(function (t) {
      window.addEventListener(t, function () { sawUserGesture = true; },
        { passive: true, capture: true });
    });
    window.addEventListener("scroll", function () {
      var y = window.scrollY;
      var dy = y - lastY;
      lastY = y;
      // A scroll with no preceding user gesture was issued by the page.
      // Programmatic scrolls are a strong automation signal on their own.
      queue.push({
        type: "scroll", t: now(), dy: dy,
        mode: sawUserGesture ? "wheel" : "programmatic"
      });
      sawUserGesture = false;
    }, { passive: true });
  }

  function attachFocus() {
    if (collect.types.indexOf("focus") === -1) return;
    ["focusin", "focusout"].forEach(function (type) {
      window.addEventListener(type, function (e) {
        queue.push({
          type: "focus", t: now(),
          state: type === "focusin" ? "focus" : "blur",
          target: targetOf(e.target)
        });
      }, { passive: true, capture: true });
    });
  }

  function attachVisibility() {
    if (collect.types.indexOf("visibility") === -1) return;
    document.addEventListener("visibilitychange", function () {
      queue.push({ type: "visibility", t: now(), state: document.visibilityState });
    });
  }

  function attachForm() {
    if (collect.types.indexOf("form") === -1) return;
    window.addEventListener("paste", function (e) {
      // The pasted CONTENT is never read.
      var id = targetOf(e.target);
      lastPasteAt[id] = now();
      queue.push({ type: "form", t: now(), target: id, action: "paste" });
    }, { passive: true, capture: true });
    window.addEventListener("submit", function (e) {
      queue.push({ type: "form", t: now(), target: targetOf(e.target), action: "submit" });
    }, { capture: true });
    // Value injection: a field's contents changed without a keystroke
    // and without a paste.
    //
    // page.fill(), element.value = "...", and send_keys-free automation
    // all land here. A person cannot change a field's contents without
    // typing into it, pasting into it, or the browser autofilling it —
    // so an input event with none of those preceding it did not come
    // from a person.
    //
    // Tracked per field rather than globally: typing in one box while
    // another is filled programmatically must still be caught.
    window.addEventListener("input", function (e) {
      var id = targetOf(e.target);
      var t = now();
      var lastKey = lastKeyAt[id] || -1e9;
      var lastPaste = lastPasteAt[id] || -1e9;

      // Autofill announces itself; treat it as its own thing.
      if (e.inputType === "insertReplacementText") {
        queue.push({ type: "form", t: t, target: id, action: "autofill" });
        return;
      }
      // 120ms covers the keydown -> input latency on a slow main thread
      // without being wide enough to launder an injection behind an
      // unrelated keystroke.
      //
      // Dictation and IME composition also reach here without a
      // preceding keydown. See policy.ReasonValueInjected — this signal
      // has a known false-positive population and must not be treated
      // as categorical.
      if (t - lastKey > 120 && t - lastPaste > 120) {
        queue.push({ type: "form", t: t, target: id, action: "injected" });
      }
    }, { passive: true, capture: true });
  }

  function srcOf(e) {
    if (e.pointerType === "touch") return "touch";
    if (e.pointerType === "pen") return "pen";
    return "mouse";
  }

  function cutPolyline(src) {
    if (pts.length === 0) return;
    queue.push({ type: "pointer", t: polylineStartedAt, src: src || "mouse", pts: pts });
    pts = [];
  }

  /* --------------------------------------------------------------
   * Batch transport
   * ------------------------------------------------------------ */
  function flush() {
    cutPolyline("mouse");
    if (!token || queue.length === 0) return;

    var events = queue;
    queue = [];

    var envelope = {
      session_token: token,
      seq: seq++,
      sent_at_ms: now(),
      page: { path: location.pathname, viewport: [window.innerWidth, window.innerHeight] },
      events: events
    };

    fetch(API + "/v1/telemetry", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer " + token
      },
      body: JSON.stringify(envelope),
      keepalive: true
    }).catch(function () {
      // Telemetry is not on the latency-critical path and loss is
      // expected (§5). Dropping the batch is correct; retrying would
      // amplify an outage into a client-side loop.
    });

    emit("flush", { seq: envelope.seq, events: events.length });
  }

  /* --------------------------------------------------------------
   * Synthetic linear injection — demo only
   * ------------------------------------------------------------ */
  // A perfectly straight, constant-velocity path: the tier-4 adversary
  // from M2 in miniature. Present so the slice is demonstrable in one
  // click; M2 drives real browsers instead.
  function injectLinear(n) {
    var t = now();
    var p = [];
    var x = 100, y = 120, step = 9;
    for (var i = 0; i < (n || 60); i++) {
      p.push([Math.round(x + i * step), Math.round(y + i * step * 0.5), i === 0 ? 0 : 16]);
    }
    queue.push({ type: "pointer", t: t, src: "mouse", pts: p });
    flush();
  }

  window.GhostTrace = {
    start: start,
    flush: flush,
    token: function () { return token; },
    onEvent: function (fn) { listeners.push(fn); },
    _injectLinear: injectLinear
  };
})();
