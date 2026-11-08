"""Optional server-side adapter with persistent deduplication and explicit errors.

Requires anthropic/flask, not CLI dependencies. Configure ANTHROPIC_API_KEY,
ANTHROPIC_WEBHOOK_SIGNING_KEY, NOBLUFF_BIN and private SQLite NOBLUFF_STATE.
Use one server process. A 120-second delivery lease permits crash recovery;
an ambiguous outbound send requires inspection, not an automatic resend.
Real signed delivery has not been validated by the offline test suite.
"""
from __future__ import annotations
import json
import os
import secrets
import shutil
import sqlite3
import subprocess
import time
from contextlib import contextmanager
import anthropic
from flask import Flask, request

client = anthropic.Anthropic()
app = Flask(__name__)


class DetectorError(RuntimeError):
    pass


class State:
    def __init__(self, path):
        self.path = path
        with self.db() as db:
            db.executescript("""
                CREATE TABLE IF NOT EXISTS deliveries(id TEXT PRIMARY KEY, state TEXT, started REAL);
                CREATE TABLE IF NOT EXISTS challenges(session TEXT, turn TEXT, text TEXT NOT NULL,
                    state TEXT NOT NULL, PRIMARY KEY(session,turn));
            """)
        os.chmod(self.path, 0o600)

    @contextmanager
    def db(self):
        db = sqlite3.connect(self.path, timeout=5)
        try:
            with db:
                yield db
        finally:
            db.close()

    def claim(self, event_id):
        with self.db() as db:
            db.execute("BEGIN IMMEDIATE")
            old = db.execute("SELECT state,started FROM deliveries WHERE id=?", (event_id,)).fetchone()
            if old and old[0] == "done":
                return "done"
            if old and time.time() - old[1] < 120:
                return "busy"
            db.execute("INSERT OR REPLACE INTO deliveries VALUES(?, 'processing', ?)", (event_id, time.time()))
            return "claimed"

    def finish(self, event_id, success):
        with self.db() as db:
            if success:
                db.execute("UPDATE deliveries SET state='done' WHERE id=?", (event_id,))
            else:
                db.execute("DELETE FROM deliveries WHERE id=?", (event_id,))

    def challenge(self, session, turn):
        with self.db() as db:
            return db.execute("SELECT text,state FROM challenges WHERE session=? AND turn=?", (session, turn)).fetchone()

    def remember(self, session, turn, text, status):
        with self.db() as db:
            db.execute("INSERT OR REPLACE INTO challenges VALUES(?,?,?,?)", (session, turn, text, status))

    def trusted_messages(self, session):
        with self.db() as db:
            return {r[0] for r in db.execute("SELECT text FROM challenges WHERE session=?", (session,))}


_state = None


def get_state():
    global _state
    if _state is None:
        path = os.environ.get("NOBLUFF_STATE")
        if not path:
            raise RuntimeError("NOBLUFF_STATE must name a private writable SQLite file")
        _state = State(path)
    return _state


def run_nobluff(events):
    binary = os.environ.get("NOBLUFF_BIN") or shutil.which("nobluff")
    if not binary:
        raise DetectorError("Set NOBLUFF_BIN to a verified local detector; no automatic npx download")
    try:
        proc = subprocess.run([binary, "check", "-"], input=json.dumps(events), capture_output=True, text=True, timeout=30)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise DetectorError(f"detector unavailable: {exc}") from exc
    if proc.returncode not in (0, 1):
        raise DetectorError(f"detector exit {proc.returncode}: {proc.stderr.strip()[:200]}")
    try:
        findings = json.loads(proc.stdout)
    except (ValueError, TypeError) as exc:
        raise DetectorError("detector output is not JSON") from exc
    if not isinstance(findings, list) or any(not isinstance(f, dict) or not isinstance(f.get("reason"), str) for f in findings):
        raise DetectorError("detector findings have an invalid schema")
    if bool(findings) != (proc.returncode == 1):
        raise DetectorError("detector exit code contradicts its findings")
    return findings


def message_text(event):
    return " ".join(b.get("text", "") for b in event.get("content") or [] if isinstance(b, dict))


def latest_turn_already_challenged(events, trusted_messages):
    latest = next((e for e in reversed(events) if e.get("type") == "user.message"), None)
    return latest is not None and message_text(latest) in trusted_messages


def handle_idle(session_id, state=None):
    state = state or get_state()
    events = [e.to_dict(mode="json") for e in client.beta.sessions.events.list(session_id=session_id)]
    idle = next((e for e in reversed(events) if e.get("type") == "session.status_idle"), None)
    if not idle or (idle.get("stop_reason") or {}).get("type") in ("requires_action", "budget_reached"):
        return "not_a_finished_turn"
    findings = run_nobluff(events)
    if not findings:
        return "clean"
    if latest_turn_already_challenged(events, state.trusted_messages(session_id)):
        app.logger.warning("NoBluff report only: session %s still has findings", session_id)
        return "reported"
    latest = next((e for e in reversed(events) if e.get("type") == "user.message"), None)
    turn = (latest or {}).get("id")
    if not turn:
        raise DetectorError("latest user turn has no event id; cannot deduplicate safely")
    previous = state.challenge(session_id, turn)
    if previous:
        if previous[1] == "sent":
            return "reported"
        raise DetectorError("challenge delivery uncertain; inspect session events before retrying the send")
    text = "[NoBluff:" + secrets.token_hex(16) + "]\n" + "\n".join(f["reason"] for f in findings)
    state.remember(session_id, turn, text, "sending")
    client.beta.sessions.events.send(session_id=session_id,
        events=[{"type": "user.message", "content": [{"type": "text", "text": text}]}])
    state.remember(session_id, turn, text, "sent")
    return "challenged"


@app.route("/webhook", methods=["POST"])
def webhook():
    try:
        event = client.beta.webhooks.unwrap(request.get_data(as_text=True), headers=dict(request.headers))
    except Exception:
        return "invalid signature", 400
    try:
        state = get_state()
        claim = state.claim(event.id)
    except Exception:
        app.logger.exception("NoBluff persistent state unavailable")
        return "state_error", 503
    if claim == "done":
        return "", 204
    if claim == "busy":
        return "delivery already processing; retry later", 409
    try:
        status = handle_idle(event.data.id, state) if event.data.type == "session.status_idled" else "ignored"
    except Exception:
        state.finish(event.id, False)
        app.logger.exception("NoBluff delivery failed; retry is permitted")
        return "detector_or_delivery_error", 503
    state.finish(event.id, True)
    return status, 200
