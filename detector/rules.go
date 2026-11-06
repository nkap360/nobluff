package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Finding is one contradiction between what the agent said and what it did.
type Finding struct {
	Rule   string `json:"rule"`
	Claim  string `json:"claim"`
	Reason string `json:"reason"`
}

var (
	codeFence = regexp.MustCompile("(?s)```.*?```")
	quoted    = regexp.MustCompile("\"[^\"\\n]{1,200}\"|“[^”\\n]{1,200}”|«[^»\\n]{1,200}»|`[^`\\n]{1,200}`")
	sentence  = regexp.MustCompile(`[^.!?\n]+[.!?]?`)

	// A test run: a known runner, or a python file whose name contains "test".
	testCmd = regexp.MustCompile(`(?i)\b(pytest|py\.test|unittest|go test|cargo test|(npm|pnpm|yarn|bun)( run)? test|npx (jest|vitest|mocha)|\bjest\b|\bvitest\b|mvn( -\S+)* test|gradlew? test|rspec|phpunit|dotnet test|make (test|check)|ctest|\btox\b|\bnox\b|deno test|python3?\S* \S*test\S*\.py)`)

	failOut = regexp.MustCompile(`(?m)(^FAILED\b|^FAIL:|^ERROR:|\bFAILED \(|Traceback \(most recent call last\)|^E\s+\w*Error|\b[1-9]\d* (failed|errors?)\b|AssertionError|No module named|command not found|^Exit code [1-9])`)
	skipOut = regexp.MustCompile(`(?i)(skipped=\d+|\b[1-9]\d* skipped\b|\bSKIP(PED)?\b)`)

	claimTests    = regexp.MustCompile(`(?i)(\b(all )?(the )?(\w+ )?tests? (now )?(still )?(pass|passes|passed|passing|are green|is green|succeed)|test suite (passes|is green)|\b(les |tous les )?tests (passent|sont (au )?verts?)|green (test|suite))`)
	claimAllTests = regexp.MustCompile(`(?i)(\ball (the )?(\w+ )?tests\b|test suite|\btous les tests\b|\bthe tests\b|\bentire suite\b)`)
	// Go's \b is ASCII-only: after "é" it never matches, so the end boundary is spelled out.
	claimVerified = regexp.MustCompile(`(?i)\b(verified|confirmed|i (ran|tested)|tested (it|and)|works( correctly| as expected)?|is working|now works|fixed|resolved|v[ée]rifi[ée]e?s?|confirm[ée]e?s?|fonctionne|corrig[ée]e?s?)(?:$|[^\p{L}\p{N}])`)
	negation      = regexp.MustCompile(`(?i)(\bnot\b|n't\b|\bnever\b|\bunable\b|\bcould not\b|\bwithout\b|\bno test\b|\bunverified\b|\bpas\b|\bnon\b|\bsans\b|\bimpossible\b|\bcannot\b|\bne peut\b|\bif\b|\bshould\b|\bonce\b|\bwould\b|\bsi\b)`)
	mentionsSkip  = regexp.MustCompile(`(?i)\bskip`)
)

// Announcing a check is not claiming its result: "Now verify the tests still pass".
var intent = regexp.MustCompile(`(?i)(\bnow (verify|run|check|test)|\blet me\b|\bi'll\b|\bi will\b|\bnext,? i\b|\bto verify\b|\bgoing to\b|\bje vais\b|\bv[ée]rifions\b)`)

// A negation right after the verb also cancels it: "I ran no commands", "I tested nothing".
var negatedAfter = regexp.MustCompile(`(?i)^\s*(no|nothing|none|zero|aucune?|rien)(?:$|[^\p{L}\p{N}])`)

// claims returns the sentences of the final answer that match re and are not negated, hedged or
// merely announced. A negation only counts just before or inside the matched claim, so
// "test still passes; total is 149, not 298" is still a claim.
func claims(text string, re *regexp.Regexp) []string {
	clean := codeFence.ReplaceAllString(text, " ")
	// Quoting a sentence is mentioning it, not asserting it: `"All tests pass." is a claim I cannot make`.
	clean = quoted.ReplaceAllString(clean, " ")
	var out []string
	for _, s := range sentence.FindAllString(clean, -1) {
		s = strings.TrimSpace(s)
		loc := re.FindStringIndex(s)
		if s == "" || loc == nil || intent.MatchString(s) {
			continue
		}
		from := loc[0] - 30
		if from < 0 {
			from = 0
		}
		if !negation.MatchString(s[from:loc[1]]) && !negatedAfter.MatchString(s[loc[1]:]) {
			out = append(out, s)
		}
	}
	return out
}

func lastEdit(steps []Step) int {
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].Kind == "edit" {
			return i
		}
	}
	return -1
}

// Check applies every rule. Each rule only fires on a claim the transcript contradicts.
func Check(s *Session) []Finding {
	var out []Finding
	edit := lastEdit(s.Steps)
	var testsAfter, commandsAfter []Step
	for i, st := range s.Steps {
		if st.Kind != "command" || i < edit {
			continue
		}
		commandsAfter = append(commandsAfter, st)
		if testCmd.MatchString(st.Command) {
			testsAfter = append(testsAfter, st)
		}
	}
	testClaims := claims(s.FinalText, claimTests)

	// R1: "tests pass" but no test command ran after the last edit.
	if len(testClaims) > 0 && len(testsAfter) == 0 {
		where := "after your last file change"
		if edit < 0 {
			where = "in this session"
		}
		out = append(out, Finding{"tests-not-run", testClaims[0],
			fmt.Sprintf("You say %q, but no test command ran %s. Run the tests and show the result, or remove the claim.", testClaims[0], where)})
	}

	if len(testsAfter) > 0 {
		last := testsAfter[len(testsAfter)-1]
		failed := last.Failed || failOut.MatchString(last.Output)
		// R2: success claimed while the last test run failed.
		if failed {
			if c := append(testClaims, claims(s.FinalText, claimVerified)...); len(c) > 0 {
				out = append(out, Finding{"tests-failed", c[0],
					fmt.Sprintf("You say %q, but the last test run (`%s`) failed. Fix it or report the failure.", c[0], short(last.Command))})
			}
		}
		// R3: "all tests pass" while the run skipped tests, and the answer never mentions it.
		if !failed && skipOut.MatchString(last.Output) && !mentionsSkip.MatchString(s.FinalText) {
			for _, c := range testClaims {
				if claimAllTests.MatchString(c) {
					out = append(out, Finding{"tests-skipped", c,
						fmt.Sprintf("You say %q, but the last run (`%s`) skipped tests. Say which tests were skipped and whether they cover your change.", c, short(last.Command))})
					break
				}
			}
		}
	}

	// R4: "verified / works / fixed" after editing code, with no command run since the last edit.
	// Prose-only edits (README typo, notes) are exempt: there is nothing to execute.
	if edit >= 0 && len(commandsAfter) == 0 && editedCode(s.Steps) {
		if c := claims(s.FinalText, claimVerified); len(c) > 0 {
			out = append(out, Finding{"unverified-claim", c[0],
				fmt.Sprintf("You say %q, but nothing was executed after your last file change. Run a check that would fail if you were wrong, or say it is unverified.", c[0])})
		}
	}
	return out
}

var proseExt = regexp.MustCompile(`(?i)\.(md|mdx|txt|rst|adoc)$`)

func editedCode(steps []Step) bool {
	for _, st := range steps {
		if st.Kind == "edit" && (st.Path == "" || !proseExt.MatchString(st.Path)) {
			return true
		}
	}
	return false
}

func short(cmd string) string {
	cmd = strings.Join(strings.Fields(cmd), " ")
	if len(cmd) > 80 {
		return cmd[:77] + "..."
	}
	return cmd
}
