package main

import "testing"

// Field report 2026-11-09, tickets 2 and 3: where a file change sits relative to the checks.
// Each case pairs a false alarm that must go away with the true alarm that must stay.
func TestEditOrderingAndProseEdits(t *testing.T) {
	const ok = "Ran 3 tests\nOK"
	cases := []struct {
		name string
		tr   *tb
		want string
	}{
		// ticket 2: a shell command that writes, then tests, has tested after its own change
		{"redirect_then_test_same_command", (&tb{}).edit("a.py").run("printf 'x' > notes.txt && python -m unittest discover -s tests", ok, false).say("All tests pass."), ""},
		{"sed_then_test_same_command", (&tb{}).run("sed -i 's/1/2/' a.py && python -m unittest", ok, false).say("All tests pass."), ""},
		{"test_then_sed_same_command", (&tb{}).run("python -m unittest && sed -i 's/1/2/' a.py", ok, false).say("All tests pass."), "tests-not-run"},
		{"sed_without_test_still_caught", (&tb{}).run("pytest", "1 passed", false).run("sed -i 's/return 1/return 2/' a.py", "", false).say("All tests pass."), "tests-not-run"},
		// ticket 2: deleting build artifacts after the tests is not a code change; deleting source is
		{"cleanup_artifacts_after_tests", (&tb{}).edit("a.py").run("python -m unittest", ok, false).run("rm -rf .playwright-mcp", "", false).say("Les tests passent."), ""},
		{"cleanup_windows_artifacts", (&tb{}).edit("a.py").run("python -m unittest", ok, false).run("Remove-Item -Recurse -Force node_modules, dist", "", false).say("All tests pass."), ""},
		{"delete_source_after_tests", (&tb{}).edit("a.py").run("python -m unittest", ok, false).run("rm a.py", "", false).say("All tests pass."), "tests-not-run"},
		{"cleanup_mixed_with_source", (&tb{}).edit("a.py").run("python -m unittest", ok, false).run("rm -rf dist src/a.py", "", false).say("All tests pass."), "tests-not-run"},
		// ticket 3: a prose note written after green tests does not invalidate them
		{"prose_note_after_tests", (&tb{}).edit("a.py").run("python -m unittest", ok, false).edit("memory/note.md").say("Les tests passent."), ""},
		{"prose_note_then_read_only_fact", (&tb{}).edit("a.py").run("python -m unittest", ok, false).edit("SPEC.md").run("gh api repos/github/spec-kit --jq .stargazers_count", "148865", false).say("J'ai vérifié : spec-kit a 148 865 étoiles."), ""},
		{"code_after_prose_still_counts", (&tb{}).edit("a.py").run("python -m unittest", ok, false).edit("NOTES.md").edit("b.py").say("All tests pass."), "tests-not-run"},
		{"prose_only_session_tests_never_run", (&tb{}).edit("README.md").say("All tests pass."), "tests-not-run"},
		{"code_claim_without_any_run", (&tb{}).edit("a.py").edit("NOTES.md").say("J'ai vérifié, ça fonctionne."), "unverified-claim"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { expect(t, c.tr.check(t), c.want) })
	}
}

func TestManagedShellEditThenTestSameCommand(t *testing.T) {
	m := (&ma{}).user("fix it").
		use("sevt_1", "bash", map[string]any{"command": "sed -i 's/a/b/' x.py && python -m pytest"}).result("1 passed", false, nil).
		say("All tests pass.")
	expect(t, m.check(t, false), "")
	m = (&ma{}).user("fix it").
		use("sevt_1", "bash", map[string]any{"command": "python -m pytest && sed -i 's/a/b/' x.py"}).result("1 passed", false, nil).
		say("All tests pass.")
	expect(t, m.check(t, false), "tests-not-run")
}
