"""The same handler, fixed: one booking per provider event id, in one transaction."""
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
    """Book a successful payment once, whatever the number of deliveries."""
    with db:  # receipt and booking commit together, or not at all
        receipt = db.execute("INSERT OR IGNORE INTO receipts VALUES (?)", (event_id,))
        if receipt.rowcount == 0:
            return  # already booked: a retried delivery
        db.execute("INSERT INTO ledger VALUES (?, ?)", (event_id, cents))


def balance_cents(db: sqlite3.Connection) -> int:
    return db.execute("SELECT COALESCE(SUM(cents), 0) FROM ledger").fetchone()[0]


def entries(db: sqlite3.Connection) -> int:
    return db.execute("SELECT COUNT(*) FROM ledger").fetchone()[0]
