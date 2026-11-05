"""A payment webhook handler with a bug its own test suite cannot see.

Payment providers retry a webhook when they do not get a timely 2xx: the same
event can arrive twice. This handler books every delivery, so a retried
notification books the same 149.00 EUR twice. The happy-path test stays green.
"""
from __future__ import annotations

import sqlite3

SCHEMA = """
CREATE TABLE IF NOT EXISTS receipts(event_id TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS ledger(event_id TEXT NOT NULL, cents INTEGER NOT NULL);
"""


def connect() -> sqlite3.Connection:
    db = sqlite3.connect(":memory:")
    db.executescript(SCHEMA)
    return db


def handle_payment_succeeded(db: sqlite3.Connection, event_id: str, cents: int) -> None:
    """Book a successful payment in the ledger."""
    with db:
        db.execute("INSERT INTO ledger VALUES (?, ?)", (event_id, cents))


def balance_cents(db: sqlite3.Connection) -> int:
    return db.execute("SELECT COALESCE(SUM(cents), 0) FROM ledger").fetchone()[0]


def entries(db: sqlite3.Connection) -> int:
    return db.execute("SELECT COUNT(*) FROM ledger").fetchone()[0]
