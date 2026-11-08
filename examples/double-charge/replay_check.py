"""Independent check of the business rule: one customer payment = one ledger entry.

It does what evidence-first asks of an agent: instead of trusting the green test,
it replays the real-world failure (the provider delivers the same event twice)
and measures the ledger.

    python replay_check.py          # the shipped handler  -> exit 1, 298.00 EUR
    python replay_check.py --fixed  # the fixed handler    -> exit 0, 149.00 EUR
"""
from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent


def load(fixed: bool):
    path = HERE / ("fixed/payments.py" if fixed else "payments.py")
    spec = importlib.util.spec_from_file_location("payments_" + ("fixed" if fixed else "shipped"), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def replay(module, deliveries: int = 2) -> tuple[int, int]:
    db = module.connect()
    try:
        for _ in range(deliveries):
            module.handle_payment_succeeded(db, "evt_042", 14900)
        return module.entries(db), module.balance_cents(db)
    finally:
        db.close()


def main(argv: list[str]) -> int:
    fixed = "--fixed" in argv
    module = load(fixed)
    n, cents = replay(module)
    label = "fixed handler" if fixed else "shipped handler"
    print(f"{label}: 1 payment of 149.00 EUR, delivered twice -> {n} ledger entr{'y' if n == 1 else 'ies'}, {cents / 100:.2f} EUR")
    if (n, cents) != (1, 14900):
        print("FAIL: the customer was charged once, the ledger booked it twice.")
        return 1
    print("PASS: one payment, one entry.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
