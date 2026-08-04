"""Promote the last run to a committed manifest under docs/results/.

Measuring and PUBLISHING are separate acts. A run in
experiments/results/ is working state and gitignored; a manifest in
docs/results/ is the record someone else will cite, so it is copied
there only when asked, only after it validates, and only from a clean
tree — a number produced from uncommitted code cannot be reproduced by
anyone including its author, and a manifest that implies otherwise is
worse than no manifest.

docs/ is the right home: the repository's rule is that it holds
artifacts about work that HAS RUN, which is exactly what this is.

    make numbers-manifest
"""
import json
import pathlib
import shutil
import subprocess
import sys

from schema import validate

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent
RUN = HERE / "results" / "numbers.json"
OUT_DIR = ROOT / "docs" / "results"


def tree_is_dirty():
    r = subprocess.run(["git", "-C", str(ROOT), "status", "--porcelain"],
                       capture_output=True, text=True)
    return bool(r.stdout.strip()), r.stdout.strip()


def main(argv):
    allow_dirty = "--allow-dirty" in argv

    if not RUN.exists():
        print(f"no run to publish at {RUN.relative_to(ROOT)} — run `make numbers` first",
              file=sys.stderr)
        return 1

    dirty, status = tree_is_dirty()
    if dirty and not allow_dirty:
        print(status, file=sys.stderr)
        print("\nthe tree is dirty — a published manifest must come from committed code.",
              file=sys.stderr)
        print("commit first, re-run `make numbers`, then publish.", file=sys.stderr)
        return 1

    doc = json.loads(RUN.read_text())

    errors = validate(doc)
    if errors:
        print("the run does not satisfy experiments/schema/numbers.schema.json:\n",
              file=sys.stderr)
        for e in errors:
            print(f"  {e}", file=sys.stderr)
        return 1

    prov = doc["provenance"]
    if prov["git"]["dirty"] and not allow_dirty:
        print("the run itself records git.dirty = true — it was measured against "
              "uncommitted code.\nre-run `make numbers` from a clean tree.", file=sys.stderr)
        return 1

    commit = prov["git"]["commit"] or "nocommit"
    name = f"numbers-{prov['generated_at'][:10]}-{commit[:8]}.json"
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(RUN, OUT_DIR / name)

    print(f"published docs/results/{name}")
    print(f"  commit    {commit}")
    print(f"  measured  {prov['generated_at']}")
    print(f"  machine   {prov['machine']['platform']}/{prov['machine']['arch']}, "
          f"{prov['machine']['cpus']} cpus")
    print(f"  mode      {prov['run']['mode']}")
    if doc["architecture"] is None:
        print("  note      number 6 was NOT measured in this run")
    if doc["false_positive_rate"] is None:
        print("  note      number 2 has no data — no human capture exists")
    if doc["absent_tiers"]:
        print(f"  note      {len(doc['absent_tiers'])} tier(s) did not run")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
