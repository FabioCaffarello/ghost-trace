"""
Statistics for the adversarial harness.

Two populations, two very different epistemic situations, and the whole
point of this file is to stop the second from being reported as if it
were the first.

BOT SIDE — reproducible and unlimited. A tier can run ten thousand
sessions overnight for free, the sessions are independent, and the
interval is tight. Detection rate is a real measurement.

HUMAN SIDE — capped by how many people will give up five minutes, and
the observations are NOT independent: sessions cluster within people.
The primary figure is therefore person-level, because the outcome is
close to a property of the person rather than of the session.

There is deliberately no code path in this file that prints a bare
session-level false-positive rate. Everything is an interval, and the
clustering adjustment is applied from an intraclass correlation
ESTIMATED FROM THE DATA rather than assumed.

Stdlib only — no numpy, no scipy. Wilson and the exact zero-event bound
are closed-form, so nothing here needs a special function.
"""
import json
import math
import pathlib
import sys
from collections import defaultdict

RESULTS = pathlib.Path(__file__).resolve().parent / "results"
Z = 1.959963984540054  # two-sided 95%


# ---------------------------------------------------------------------
# intervals
# ---------------------------------------------------------------------

def wilson(successes, n, z=Z):
    """Wilson score interval. Behaves sanely at 0 and at small n, where
    the normal approximation produces intervals that include impossible
    values or collapse to zero width."""
    if n <= 0:
        return (0.0, 1.0)
    p = successes / n
    d = 1 + z * z / n
    centre = (p + z * z / (2 * n)) / d
    half = (z / d) * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n))
    return (max(0.0, centre - half), min(1.0, centre + half))


def zero_event_upper(n, alpha=0.05):
    """Exact one-sided upper bound with zero observed events.

    1 - alpha^(1/n). The familiar 3/n 'rule of three' is its large-n
    approximation; both are reported because 3/n is the number people
    recognise and this is the number that is correct at small n."""
    if n <= 0:
        return 1.0
    return 1 - alpha ** (1 / n)


# ---------------------------------------------------------------------
# clustering
# ---------------------------------------------------------------------

def icc_anova(clusters):
    """One-way ANOVA estimate of the intraclass correlation for a binary
    outcome.

    clusters: {key: [0/1, ...]}

    This is the number an earlier draft of the plan simply assumed at
    0.2. It must be estimated, because the assumption contradicted the
    very argument it sat next to: if false positives concentrate in
    people with atypical motor patterns, then being flagged is close to
    a property of the person, and rho is high — which is exactly what
    shrinks the effective sample size.

    Returns None when there is too little structure to estimate."""
    groups = [v for v in clusters.values() if len(v) > 0]
    k = len(groups)
    if k < 2:
        return None

    n_i = [len(g) for g in groups]
    N = sum(n_i)
    if N <= k:
        return None

    y_i = [sum(g) for g in groups]
    p_bar = sum(y_i) / N
    p_i = [y / n for y, n in zip(y_i, n_i)]

    msb = sum(n * (p - p_bar) ** 2 for n, p in zip(n_i, p_i)) / (k - 1)
    msw = sum(n * p * (1 - p) for n, p in zip(n_i, p_i)) / (N - k)

    m0 = (N - sum(n * n for n in n_i) / N) / (k - 1)
    denom = msb + (m0 - 1) * msw
    if denom <= 0:
        return 0.0
    return max(0.0, min(1.0, (msb - msw) / denom))


def design_effect(rho, mean_cluster_size):
    return 1 + (mean_cluster_size - 1) * rho


# ---------------------------------------------------------------------
# loading
# ---------------------------------------------------------------------

def load(path):
    p = RESULTS / path
    if not p.exists():
        return []
    rows = []
    with open(p) as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def flagged(row):
    """A session is flagged if the enforcing decision is not allow.

    In monitor mode the returned decision is always allow, so the shadow
    is what enforce would have done — that is the number under test."""
    d = row.get("shadow_decision") or row.get("decision")
    return d != "allow"


# ---------------------------------------------------------------------
# reports
# ---------------------------------------------------------------------

def report_bots(rows):
    print("\n" + "=" * 74)
    print("BOT SIDE — detection rate per adversarial tier")
    print("=" * 74)
    if not rows:
        print("  no bot sessions recorded")
        return

    by_cohort = defaultdict(list)
    for r in rows:
        by_cohort[r["cohort"]].append(r)

    print(f"\n  {'tier':<30} {'n':>4}  {'detected':>8}  {'rate':>6}  {'95% CI':>16}  {'mean score':>10}")
    print("  " + "-" * 84)
    for cohort in sorted(by_cohort):
        rs = by_cohort[cohort]
        n = len(rs)
        det = sum(1 for r in rs if flagged(r))
        lo, hi = wilson(det, n)
        mean_score = sum(r["score"] for r in rs) / n
        print(
            f"  {cohort:<30} {n:>4}  {det:>8}  {det/n:>6.1%}  "
            f"[{lo:>5.1%}, {hi:>5.1%}]  {mean_score:>10.3f}"
        )

    # Evidence volume separates the two evasion modes and the summary
    # is misleading without it: a tier can score zero because it looks
    # human, or because it produced nothing to look at.
    print(f"\n  {'tier':<30} {'mean events':>12} {'mean conf':>10}   evasion mode")
    print("  " + "-" * 84)
    for cohort in sorted(by_cohort):
        rs = by_cohort[cohort]
        n = len(rs)
        ev = sum(r["events"] for r in rs) / n
        cf = sum(r["confidence"] for r in rs) / n
        if cf < 0.4:
            mode = "by absence — too little evidence to judge"
        elif sum(1 for r in rs if flagged(r)) / n > 0.5:
            mode = "none — detected"
        else:
            mode = "by mimicry — evidence present, looks human"
        print(f"  {cohort:<30} {ev:>12.0f} {cf:>10.3f}   {mode}")


def report_humans(rows):
    print("\n" + "=" * 74)
    print("HUMAN SIDE — false positives")
    print("=" * 74)
    if not rows:
        print("""
  No human sessions recorded yet.

  This is the binding constraint on the entire result, and it is
  calendar-bound rather than effort-bound: it needs people, and no
  amount of engineering time substitutes. See harness/README.md.
""")
        return

    arm_a = [r for r in rows if r.get("arm") == "A"]
    arm_b = [r for r in rows if r.get("arm") == "B"]

    if arm_b:
        report_arm_b(arm_b)
    if arm_a:
        report_arm_a(arm_a)


def report_arm_b(rows):
    """Arm B — breadth. Many people, few sessions each. The FPR arm."""
    print("\n  ARM B — false-positive breadth (many people, few sessions each)")
    print("  " + "-" * 70)

    by_person = defaultdict(list)
    for r in rows:
        by_person[r["participant"]].append(r)

    people = len(by_person)
    sessions = len(rows)

    # ---- primary figure: the person is the unit of analysis ----
    #
    # This sidesteps the clustering problem instead of correcting for
    # it, is directly interpretable, and cannot be inflated by running
    # more sessions per person.
    people_flagged = sum(1 for rs in by_person.values() if any(flagged(r) for r in rs))
    lo, hi = wilson(people_flagged, people)

    print(f"\n  PRIMARY  {people_flagged} of {people} people were falsely flagged at least once")
    print(f"           95% CI on the proportion of people: [{lo:.1%}, {hi:.1%}]")
    print(f"           ({sessions} sessions, {sessions/people:.1f} per person)")

    # ---- secondary: session-level, cluster-adjusted ----
    binary = {p: [1 if flagged(r) else 0 for r in rs] for p, rs in by_person.items()}
    rho = icc_anova(binary)
    mean_m = sessions / people

    sess_flagged = sum(1 for r in rows if flagged(r))

    print("\n  SECONDARY  session-level, adjusted for clustering within people")
    if rho is None:
        print("             rho not estimable (need >= 2 people)")
        n_eff = people
    else:
        deff = design_effect(rho, mean_m)
        n_eff = sessions / deff if deff > 0 else people
        print(f"             rho (estimated from data) = {rho:.3f}")
        print(f"             design effect = {deff:.2f}   effective n = {n_eff:.1f} of {sessions}")

    eff_succ = sess_flagged * (n_eff / sessions) if sessions else 0
    lo, hi = wilson(eff_succ, n_eff)
    print(f"             {sess_flagged}/{sessions} sessions flagged")
    print(f"             95% CI, cluster-adjusted: [{lo:.1%}, {hi:.1%}]")

    naive_lo, naive_hi = wilson(sess_flagged, sessions)
    print(f"             (unadjusted would report [{naive_lo:.1%}, {naive_hi:.1%}] — too narrow)")

    if sess_flagged == 0:
        print(f"\n  ZERO EVENTS OBSERVED")
        print(f"             exact 95% upper bound on FPR: {zero_event_upper(n_eff):.1%}")
        print(f"             rule-of-three approximation (3/n_eff): {3/n_eff:.1%}"
              if n_eff > 0 else "")
        print(f"             a production anti-bot system needs ~0.1%, which needs")
        print(f"             roughly 3,000 independent sessions. This is not that.")


def report_arm_a(rows):
    """Arm A — depth. Few people, many sessions, crossed conditions.

    Reports condition effect sizes and deliberately reports NO false
    positive rate: three people cannot support one, and printing a
    number here would be the exact false precision this file exists to
    prevent."""
    print("\n  ARM A — condition sensitivity (few people, many sessions)")
    print("  " + "-" * 70)
    print("  Reports effect sizes only. No FPR is computed from Arm A:")
    print("  a handful of people cannot support a population rate.")

    by_cond = defaultdict(list)
    for r in rows:
        by_cond[r.get("condition", "unspecified")].append(r)

    print(f"\n  {'condition':<28} {'n':>4} {'mean score':>11} {'mean conf':>10} {'flagged':>8}")
    print("  " + "-" * 70)
    for cond in sorted(by_cond):
        rs = by_cond[cond]
        n = len(rs)
        print(
            f"  {cond:<28} {n:>4} "
            f"{sum(r['score'] for r in rs)/n:>11.3f} "
            f"{sum(r['confidence'] for r in rs)/n:>10.3f} "
            f"{sum(1 for r in rs if flagged(r)):>8}"
        )

    # Habituation is the effect most likely to matter: the 30th visit to
    # a form produces fast, ballistic, low-correction movement that looks
    # markedly more synthetic than the 1st. If that is large, the system
    # degrades against its most loyal users.
    by_person_visit = defaultdict(list)
    for r in rows:
        if r.get("visit") is not None:
            by_person_visit[r["participant"]].append((r["visit"], r["score"]))

    if by_person_visit:
        print("\n  HABITUATION — score drift by visit index, within person")
        for person, pairs in sorted(by_person_visit.items()):
            pairs.sort()
            if len(pairs) < 4:
                continue
            third = max(1, len(pairs) // 3)
            early = sum(s for _, s in pairs[:third]) / third
            late = sum(s for _, s in pairs[-third:]) / third
            arrow = "↑" if late > early else "↓" if late < early else "="
            print(f"    {person:<20} first-third {early:.3f}  ->  last-third {late:.3f}  {arrow}")


def caveats():
    print("\n" + "=" * 74)
    print("WHY THE FALSE-POSITIVE RATE IS THE WEAK METRIC")
    print("=" * 74)
    print("""
  1. SAMPLE SIZE. Bot tiers are unlimited; people are not. A production
     anti-bot system operates near 0.1% FPR, which needs roughly 3,000
     independent observations to bound. This study reaches two orders of
     magnitude short of that, and no amount of engineering closes it.

  2. CLUSTERING. Sessions within one person are not independent. If
     being flagged is close to a property of the person — and the whole
     argument below says it is — then effective sample size approaches
     the NUMBER OF PEOPLE, no matter how many sessions each contributes.
     Eight people running a thousand sessions each still gives n_eff ~ 8.

  3. DIRECTIONAL BIAS, which no interval captures. The humans most
     likely to be falsely flagged are those whose motor patterns differ
     most from the norm: tremor, motor impairment, switch or eye-tracking
     input, non-dominant-hand use, very old and very young users. A
     volunteer pool drawn from one person's social graph systematically
     under-samples exactly that population.

     So the figure is not merely imprecise. It is OPTIMISTIC IN A KNOWN
     DIRECTION, and that direction carries the highest human cost —
     blocking a human is far worse than admitting a bot.

  Points 2 and 3 are the same fact seen from two sides. If false
  positives were spread evenly across people, rho would be low and
  pooled session counts would be honest. They are not spread evenly,
  which is simultaneously why rho is high, why the person is the right
  unit of analysis, and why the sample is biased.
""")


def selftest():
    """Run the statistics against the synthetic fixture.

    The fixture is NOT data. It exists so the person-level path, the ICC
    estimator and the design-effect adjustment are exercised before
    twenty real people spend five minutes each — shipping untested
    statistics to a study you can only run once would be careless."""
    path = pathlib.Path(__file__).resolve().parent / "testdata" / "synthetic_human_sessions.jsonl"
    if not path.exists():
        print("fixture missing; run: python3 testdata/make_synthetic_human.py")
        return 1
    rows = [json.loads(l) for l in path.read_text().splitlines() if l.strip()]
    print("\n*** SELF-TEST on SYNTHETIC data — these are not measurements ***")
    report_humans(rows)
    return 0


def main():
    if "--selftest" in sys.argv:
        return selftest()

    bots = load("sessions.jsonl")
    humans = load("human_sessions.jsonl")

    print("\nGhost Trace — M2 adversarial harness")
    print(f"bot sessions: {len(bots)}   human sessions: {len(humans)}")

    report_bots(bots)
    report_humans(humans)
    caveats()

    absent = RESULTS / "absent_tiers.txt"
    if absent.exists():
        names = [l.strip() for l in absent.read_text().splitlines() if l.strip()]
        if names:
            print("=" * 74)
            print("TIERS THAT DID NOT RUN")
            print("=" * 74)
            for n in names:
                print(f"  {n}")
            print("\n  A tier that did not run is not a tier that found nothing.")
            print()
    return 0


if __name__ == "__main__":
    sys.exit(main())
