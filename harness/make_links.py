"""Generate participant links for the capture study.

    python3 harness/make_links.py --host 192.168.1.20 --arm B --people 20

Arm B is breadth: many people, few sessions each. Its precision is
governed by PERSON COUNT alone, so the ask is "five minutes each from as
many people as possible" — something you can send to a group chat —
rather than "thirty sessions from a friend".

Arm A is depth: a few people, many sessions, conditions crossed. It
answers whether habituation and input device move the score, and it
reports no false-positive rate at all.
"""
import argparse
import secrets

ARM_B_CONDITIONS = ["mouse-desktop", "trackpad-desktop", "touch-mobile"]
ARM_A_CONDITIONS = [
    "mouse-alert", "mouse-tired", "mouse-distracted",
    "trackpad-alert", "trackpad-tired",
    "touch-mobile", "mouse-reduced-motion",
]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--host", required=True, help="host:port reachable by participants")
    ap.add_argument("--arm", choices=["A", "B"], default="B")
    ap.add_argument("--people", type=int, default=20)
    ap.add_argument("--sessions", type=int, default=4)
    args = ap.parse_args()

    base = args.host if args.host.startswith("http") else f"http://{args.host}"
    conds = ARM_A_CONDITIONS if args.arm == "A" else ARM_B_CONDITIONS

    print(f"\n# Arm {args.arm}: {args.people} people x {args.sessions} sessions")
    print("# Participant codes are pseudonyms. Do not map them to names in\n"
          "# anything you commit; the point is that the study never needs one.\n")

    for i in range(1, args.people + 1):
        code = f"p{i:02d}-{secrets.token_hex(2)}"
        print(f"## participant {i}")
        for v in range(1, args.sessions + 1):
            cond = conds[(v - 1) % len(conds)]
            print(f"{base}/?p={code}&arm={args.arm}&c={cond}&v={v}")
        print()

    print("""# What to tell participants:
#   - Only HOW THE POINTER MOVES is recorded. Not what you type —
#     no key events are collected at all.
#   - No canvas, WebGL, font or audio fingerprinting.
#   - Nothing persistent is written to your browser.
#   - Put anything in the form fields; the content is never read.
#   - Use the page as you normally would, then sign in. One link per go.""")


if __name__ == "__main__":
    main()
