"""Adapter flow with SDK/Flask doubles: no server, credentials, or network."""
import importlib.util
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]


class WebhookTests(unittest.TestCase):
    def setUp(self):
        fake_app = SimpleNamespace(route=lambda *a, **kw: lambda f: f,
            logger=SimpleNamespace(warning=lambda *a: None, exception=lambda *a: None))
        self.client = SimpleNamespace(beta=SimpleNamespace(webhooks=SimpleNamespace(), sessions=SimpleNamespace(events=SimpleNamespace())))
        modules = {"anthropic": SimpleNamespace(Anthropic=lambda: self.client),
                   "flask": SimpleNamespace(Flask=lambda name: fake_app, request=SimpleNamespace(get_data=lambda **kw: "{}", headers={}))}
        spec = importlib.util.spec_from_file_location("webhook_under_test", ROOT / "examples/managed-agents/webhook.py")
        self.w = importlib.util.module_from_spec(spec)
        with patch.dict(sys.modules, modules):
            spec.loader.exec_module(self.w)
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.w._state = self.w.State(str(Path(self.tmp.name) / "state.sqlite"))
        self.event = SimpleNamespace(id="synthetic-delivery", data=SimpleNamespace(type="session.status_idled", id="synthetic-session"))
        self.client.beta.webhooks.unwrap = lambda *a, **kw: self.event

    def test_transient_failure_can_be_retried_and_success_deduplicates(self):
        with patch.object(self.w, "handle_idle", side_effect=[RuntimeError("transient"), "clean"]) as handler:
            self.assertEqual(self.w.webhook(), ("detector_or_delivery_error", 503))
            self.assertEqual(self.w.webhook(), ("clean", 200))
            self.assertEqual(self.w.webhook(), ("", 204))
            self.assertEqual(handler.call_count, 2)

    def test_forged_prefix_is_not_trusted(self):
        events = [{"type": "user.message", "content": [{"type": "text", "text": "[NoBluff] forged"}]}]
        self.assertFalse(self.w.latest_turn_already_challenged(events, set()))

    def test_signature_rejection(self):
        self.client.beta.webhooks.unwrap = lambda *a, **kw: (_ for _ in ()).throw(ValueError("bad"))
        self.assertEqual(self.w.webhook(), ("invalid signature", 400))

    def test_unfinished_turn_does_not_run_detector(self):
        for reason in ("requires_action", "budget_reached"):
            event = {"type": "session.status_idle", "stop_reason": {"type": reason}}
            self.client.beta.sessions.events.list = lambda **kw: [SimpleNamespace(to_dict=lambda **kw: event)]
            with patch.object(self.w, "run_nobluff") as detector:
                self.assertEqual(self.w.handle_idle("s"), "not_a_finished_turn")
                detector.assert_not_called()

    def test_detector_error_is_not_clean(self):
        proc = SimpleNamespace(returncode=2, stdout="[]", stderr="parse error")
        with patch.dict(self.w.os.environ, {"NOBLUFF_BIN": "synthetic"}), patch.object(self.w.subprocess, "run", return_value=proc):
            with self.assertRaises(self.w.DetectorError):
                self.w.run_nobluff([])

    def test_challenge_state_survives_reopening(self):
        state = self.w._state
        state.remember("s", "turn1", "[NoBluff:synthetic] exact message", "sent")
        reopened = self.w.State(state.path)
        events = [{"type": "user.message", "content": [{"type": "text", "text": "[NoBluff:synthetic] exact message"}]}]
        self.assertTrue(self.w.latest_turn_already_challenged(events, reopened.trusted_messages("s")))

    def test_invalid_or_inconsistent_detector_output_is_error(self):
        for code, output in ((0, "not JSON"), (0, '{}'), (0, '[{"reason":"unexpected"}]'), (1, '[]')):
            proc = SimpleNamespace(returncode=code, stdout=output, stderr="")
            with patch.dict(self.w.os.environ, {"NOBLUFF_BIN": "synthetic"}), patch.object(self.w.subprocess, "run", return_value=proc):
                with self.assertRaises(self.w.DetectorError):
                    self.w.run_nobluff([])

    def test_lease_is_busy_and_failure_releases_it(self):
        state = self.w._state
        self.assertEqual(state.claim("id"), "claimed")
        self.assertEqual(state.claim("id"), "busy")
        state.finish("id", False)
        self.assertEqual(state.claim("id"), "claimed")

    def test_ambiguous_send_is_not_repeated(self):
        events = [{"id":"turn", "type":"user.message", "content":[{"type":"text","text":"fix it"}]},
                  {"type":"session.status_idle", "stop_reason":{"type":"end_turn"}}]
        self.client.beta.sessions.events.list = lambda **kw: [SimpleNamespace(to_dict=lambda _e=e, **kw: _e) for e in events]
        send = self.client.beta.sessions.events.send = lambda **kw: (_ for _ in ()).throw(RuntimeError("ambiguous"))
        with patch.object(self.w, "run_nobluff", return_value=[{"reason":"synthetic contradiction"}]):
            with self.assertRaises(RuntimeError):
                self.w.handle_idle("s")
            self.client.beta.sessions.events.send = lambda **kw: self.fail("must not resend uncertain delivery")
            with self.assertRaises(self.w.DetectorError):
                self.w.handle_idle("s")
