"""Exercise analyze.py's clustering maths on a SYNTHETIC human sample.

This is a test fixture for the statistics, not data. It is written to a
separate file that analyze.py does not read, so it can never be mistaken
for capture. Its only job is to prove the person-level path, the ICC
estimator and the design-effect adjustment behave sanely before twenty
real people spend five minutes each.
"""
import json, random, pathlib
random.seed(7)
out = []
# Arm B: 20 people x 4 sessions. Flagging is mostly a PROPERTY OF THE
# PERSON — 3 of 20 are "atypical" and flag often — which is the high-rho
# structure the estimator must recover.
atypical = {"p03", "p11", "p17"}
for i in range(1, 21):
    pid = f"p{i:02d}"
    base = 0.75 if pid in atypical else 0.02
    for v in range(1, 5):
        flagged = random.random() < base
        out.append({
            "participant": pid, "arm": "B",
            "condition": random.choice(["mouse-desktop", "trackpad-desktop", "touch-mobile"]),
            "visit": v, "evaluation_id": f"ev_{pid}_{v}",
            "decision": "allow",
            "shadow_decision": "challenge" if flagged else "allow",
            "score": round(random.uniform(0.5, 0.9) if flagged else random.uniform(0.0, 0.3), 3),
            "confidence": round(random.uniform(0.6, 1.0), 3),
            "events": random.randint(80, 400), "duration_ms": random.randint(4000, 20000),
            "at": "2026-08-03T00:00:00Z",
        })
# Arm A: 3 people x 30 sessions, habituation drift built in.
for pid in ("p01", "p02", "p03"):
    for v in range(1, 31):
        drift = v / 30 * 0.25
        out.append({
            "participant": pid, "arm": "A",
            "condition": random.choice(["mouse-alert", "trackpad-tired", "mouse-reduced-motion"]),
            "visit": v, "evaluation_id": f"ev_a_{pid}_{v}",
            "decision": "allow", "shadow_decision": "allow",
            "score": round(min(1.0, random.uniform(0.05, 0.25) + drift), 3),
            "confidence": round(random.uniform(0.7, 1.0), 3),
            "events": random.randint(100, 500), "duration_ms": random.randint(5000, 25000),
            "at": "2026-08-03T00:00:00Z",
        })
p = pathlib.Path("testdata/synthetic_human_sessions.jsonl")
p.write_text("".join(json.dumps(r) + "\n" for r in out))
print(f"wrote {len(out)} synthetic rows to {p}")
