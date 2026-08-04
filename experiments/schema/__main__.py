"""Entry point so `python3 -m schema --selftest` works on a package."""
import sys

from . import selftest

if __name__ == "__main__":
    if "--selftest" in sys.argv:
        sys.exit(selftest())
    print(__doc__)
    sys.exit(2)
