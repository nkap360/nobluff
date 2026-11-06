"""NoBluff for Claude Managed Agents: a server-side Stop hook.

When a Managed Agents session goes idle at the end of a turn, Anthropic POSTs a
`session.status_idled` webhook. This server lists the session's events, runs the NoBluff
detector on them, and, if the final answer claims something the events contradict
("tests pass" with no test run after the last edit, "fixed" with a failing last run...),
sends the agent back to work with the reason. Like the Claude Code hook, it pushes back once
per turn, then only reports, so it can never loop.

    pip install anthropic flask
    export ANTHROPIC_API_KEY=...                  # or an `ant auth login` profile
    export ANTHROPIC_WEBHOOK_SIGNING_KEY=whsec_...
    export NOBLUFF_BIN=/path/to/nobluff           # npx nobluff also works, see run_nobluff()
    flask --app webhook run --port 8080           # then register the URL in the Console

Status: the detector side is validated on a live session (detector/testdata/managed_live_session.json);
this webhook server itself has not yet received a real webhook delivery.
"""
from __future__ import annotations

import json
import os
import shutil
import subprocess

import anthropic
from flask import Flask, request

client = anthropic.Anthropic()  # reads ANTHROPIC_WEBHOOK_SIGNING_KEY for webhooks.unwrap()
app = Flask(__name__)
seen_event_ids: set[str] = set()  # dedupe webhook retries (use a real store in production)
MARKER = "[NoBluff]"


def run_nobluff(events: list[dict]) -> list[dict]:
    """Run `nobluff check -` on the session events; return the findings (empty if none)."""
    binary = os.environ.get("NOBLUFF_BIN") or shutil.which("nobluff")
    cmd = [binary, "check", "-"] if binary else ["npx", "--yes", "nobluff", "check", "-"]
    proc = subprocess.run(cmd, input=json.dumps(events), capture_output=True, text=True, timeout=30)
    if proc.returncode not in (0, 1):  # 2 = could not read; never block the agent on our own error
        app.logger.warning("nobluff failed: %s", proc.stderr.strip())
        return []
    return json.loads(proc.stdout or "[]")


def latest_turn_already_challenged(events: list[dict]) -> bool:
    """True if we already pushed back in the current turn (the hook's one-correction rule)."""
    for e in reversed(events):
        if e.get("type") == "user.message":
            text = " ".join(b.get("text", "") for b in e.get("content") or [] if isinstance(b, dict))
            return text.startswith(MARKER)
    return False


@app.route("/webhook", methods=["POST"])
def webhook():
    try:
        event = client.beta.webhooks.unwrap(request.get_data(as_text=True), headers=dict(request.headers))
    except Exception:
        return "invalid signature", 400
    if event.id in seen_event_ids:
        return "", 204
    seen_event_ids.add(event.id)
    if event.data.type == "session.status_idled":
        handle_idle(event.data.id)
    return "", 204


def handle_idle(session_id: str) -> str:
    """Judge the turn that just ended; returns what was done (for logs and tests)."""
    # mode="json": events carry datetimes (processed_at) that json.dumps cannot encode otherwise
    events = [e.to_dict(mode="json") for e in client.beta.sessions.events.list(session_id=session_id)]

    # Only judge a finished turn: an idle session can also be waiting for a tool approval,
    # a custom tool result, or be paused at its budget, and a user.message would then be rejected.
    # The two non-final stop reasons below are the documented ones; anything else is a finished turn.
    idle = next((e for e in reversed(events) if e.get("type") == "session.status_idle"), None)
    if not idle or (idle.get("stop_reason") or {}).get("type") in ("requires_action", "budget_reached"):
        return "not a finished turn"

    findings = run_nobluff(events)
    if not findings:
        return "clean"
    reasons = "\n".join(f"- {f['reason']}" for f in findings)
    if latest_turn_already_challenged(events):
        app.logger.warning("NoBluff (report only, already challenged) session %s:\n%s", session_id, reasons)
        return "reported"
    client.beta.sessions.events.send(
        session_id=session_id,
        events=[{
            "type": "user.message",
            "content": [{"type": "text", "text": (
                f"{MARKER} Your last answer claims something this session's events contradict:\n{reasons}\n"
                "Run the check and show its result, or state plainly that it is not verified."
            )}],
        }],
    )
    return "challenged"
