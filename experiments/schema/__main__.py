"""Entry point for the schema package.

`python3 -m schema --selftest` proves the validator against the fixture
corpus. `python3 -m schema FILE...` validates each file against
numbers.schema.json — `make experiments-check` runs it over
docs/results/numbers-*.json, because a published corpus not held to the
contract it advertises is how a manifest with no seed sat committed for
a week while the schema said that could not happen.
"""
import json
import sys

from . import selftest, validate_numbers

if __name__ == "__main__":
    if "--selftest" in sys.argv:
        sys.exit(selftest())
    paths = [a for a in sys.argv[1:] if not a.startswith("-")]
    if not paths:
        print(__doc__)
        sys.exit(2)
    bad = 0
    for p in sorted(paths):
        with open(p, encoding="utf-8") as fh:
            errors = validate_numbers(json.load(fh))
        if errors:
            bad += 1
            print(f"  {p}: does NOT satisfy numbers.schema.json")
            for e in errors:
                print(f"    {e}")
        else:
            print(f"  {p}: ok")
    sys.exit(1 if bad else 0)
