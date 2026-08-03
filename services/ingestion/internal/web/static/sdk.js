/*
 * Ghost Trace browser SDK — M1.
 *
 * Collects pointer geometry only. No canvas, WebGL, font enumeration or
 * audio fingerprinting: those identify the browser rather than the
 * behaviour and are out of scope per integration-contract.md §0.
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
