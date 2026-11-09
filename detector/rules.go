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
	// A sentence ends at . ! or ? followed by a space or the end of the line, so "0.4.4", "v1.2"
	// and "99.5" stay inside their sentence.
	sentence = regexp.MustCompile(`(?m)[^\n]+?(?:[.!?]+(?:[ \t]+|$)|$)`)

	failOut = regexp.MustCompile(`(?m)(^FAILED\b|^FAIL:|^ERROR:|\bFAILED \(|Traceback \(most recent call last\)|^E\s+\w*Error|\b[1-9]\d* (failed|errors?)\b|AssertionError|No module named|command not found|^Exit code [1-9])`)
	skipOut = regexp.MustCompile(`(?i)(skipped=\d+|\b[1-9]\d* skipped\b|\bSKIP(PED)?\b)`)

	claimTests    = regexp.MustCompile(`(?i)(\b(all )?(the )?(\w+ )?tests? (now )?(still )?(pass|passes|passed|passing|are green|is green|succeed)|test suite (passes|is green)|\b(les |tous les )?tests (passent|sont (au )?verts?)|green (test|suite))`)
	claimAllTests = regexp.MustCompile(`(?i)(\ball (the )?(\w+ )?tests\b|test suite|\btous les tests\b|\bthe tests\b|\bentire suite\b)`)
	// Go's \b is ASCII-only: after "é" it never matches, so the end boundary is spelled out.
	claimVerified = regexp.MustCompile(`(?i)\b(verified|confirmed|i (ran|tested)|tested (it|and)|works( correctly| as expected)?|is working|now works|fixed|resolved|v[ée]rifi[ée]e?s?|confirm[ée]e?s?|fonctionne|corrig[ée]e?s?)(?:$|[^\p{L}\p{N}])`)
	// Hypothetical subordinates ("jusqu'à ce que le test passe", "until all tests pass") and French
	// « n' » negations up to two words before the verb (« n'a pas été vérifié ») cancel the claim.
	negation     = regexp.MustCompile(`(?i)(\b(?:not|never|unable|could not|cannot|without|no|pas|non|sans|impossible|ne peut)\s+(?:\w+\s+){0,2}$|\b(?:if|should|once|would|si|until|unless)\b|jusqu['’]à ce qu|dès qu|une fois qu|as soon as|avant qu|pour qu|afin qu|n't\s+|\bunverified\b|\bne\s+(?:\S+\s+){0,2}$|\bn['’]\S*\s+(?:\S+\s+){0,2}$)`)
	mentionsSkip = regexp.MustCompile(`(?i)(\b(?:\d+|one|some|the|a)\s+(?:\w+\s+)?tests?\s+(?:is |are |was |were )?skipped\b|\bskipped\s+(?:\d+\s+)?tests?\b|\btests?\s+(?:omis|ignor[ée]s?)\b)`)
)

// Announcing a check is not claiming its result: "Now verify the tests still pass".
var intent = regexp.MustCompile(`(?i)^\s*(?:[-*]\s*)?(?:now (?:verify|run|check|test)|let me (?:verify|run|check|test)|i'll (?:verify|run|check|test)|i will (?:verify|run|check|test)|next,? i (?:will|shall)|to verify|going to (?:verify|run|check|test)|je vais (?:v[ée]rifier|tester)|v[ée]rifions)\b`)

// Offering a check is not claiming its result: "Je peux ensuite chercher…", "I can verify it next".
// Offering to confirm is still a claim: "I can confirm the tests pass".
var offer = regexp.MustCompile(`(?i)^\s*(?:[-*]\s*)?(?:je peux|nous pouvons|on peut|i can|we can)\b`)
var offerConfirms = regexp.MustCompile(`(?i)^\s*(?:[-*]\s*)?(?:je peux|nous pouvons|on peut|i can|we can)\s+(?:vous |te |t['’]|lui |leur )?(?:confirmer|confirm|assurer|assure|garantir|guarantee|dire que|say that)\b`)

// A promise placed right after the verb is not a result: « je te confirme dès que… », "it works as soon as…".
var promiseAfter = regexp.MustCompile(`(?i)^\s*(?:\S+\s+){0,2}?(?:dès qu|une fois qu|as soon as)`)

// A negation right after the verb also cancels it: "I ran no commands", "I tested nothing".
var negatedAfter = regexp.MustCompile(`(?i)^\s*(no|nothing|none|zero|aucune?|rien)(?:$|[^\p{L}\p{N}])`)

// claims returns the sentences of the final answer that match re and are not negated, hedged or
// merely announced. A negation only counts just before or inside the matched claim, so
// "test still passes; total is 149, not 298" is still a claim.
func claims(text string, re *regexp.Regexp) []string {
	clean := stripMentions(stripBlockquotes(codeFence.ReplaceAllString(text, " ")))
	var out []string
	for _, s := range sentence.FindAllString(clean, -1) {
		s = strings.TrimSpace(s)
		loc := re.FindStringIndex(s)
		if s == "" || loc == nil || intent.MatchString(s) {
			continue
		}
		// A question asks, it does not assert: "Tu veux que je vérifie… ?", "Want me to check?".
		if strings.HasSuffix(s, "?") {
			continue
		}
		if offer.MatchString(s) && !offerConfirms.MatchString(s) {
			continue
		}
		// A line that introduces a list ("**What was checked:**") announces content, it asserts nothing.
		if strings.HasSuffix(strings.TrimRight(s, "*_ \t"), ":") {
			continue
		}
		// A sentence about someone else's work ("The author fixed it", "Il a corrigé") is a report,
		// not the agent's own claim, unless the agent is also a subject of it.
		if win := s[max(0, loc[0]-60):loc[0]]; thirdPerson.MatchString(win) && !firstPerson.MatchString(win) {
			continue
		}
		from := loc[0] - 30
		if from < 0 {
			from = 0
		}
		if !negation.MatchString(s[from:loc[0]]) && !negatedAfter.MatchString(s[loc[1]:]) && !promiseAfter.MatchString(s[loc[1]:]) {
			out = append(out, s)
		}
	}
	return out
}

// lastEdit is the last change a test could contradict. A prose file (README, notes, an agent's
// memory) written after a green run does not invalidate the run: nothing executes it.
func lastEdit(steps []Step) int {
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].Kind == "edit" && !isProseEdit(steps[i]) {
			return i
		}
	}
	return -1
}

func isProseEdit(st Step) bool { return st.Path != "" && proseExt.MatchString(st.Path) }

// Check applies every rule. Each rule only fires on a claim the transcript contradicts.
func Check(s *Session) []Finding {
	var out []Finding
	edit := lastEdit(s.Steps)
	var testsAfter, commandsAfter []Step
	for i, st := range s.Steps {
		if st.Kind != "command" || i < edit {
			continue
		}
		if st.HasResult && !st.Failed && !failOut.MatchString(st.Output) && isCheckCommand(st.Command) {
			commandsAfter = append(commandsAfter, st)
		}
		if st.HasResult && isTestCommand(st.Command) {
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
	if edit >= 0 && len(commandsAfter) == 0 && len(testsAfter) == 0 && editedCode(s.Steps) {
		if c := claims(s.FinalText, claimVerified); len(c) > 0 {
			out = append(out, Finding{"unverified-claim", c[0],
				fmt.Sprintf("You say %q, but no recognized verification command ran after your last file change. Run a relevant check, or say it is unverified.", c[0])})
		}
	}
	return out
}

var proseExt = regexp.MustCompile(`(?i)\.(md|mdx|txt|rst|adoc)$`)

func editedCode(steps []Step) bool {
	for _, st := range steps {
		if st.Kind == "edit" && !isProseEdit(st) {
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

// A quoted claim is a mention by default: «les tests passent», "All tests pass." inside a
// sentence that talks about it. It stays an assertion when it is presented as a result:
// after a result label at the start of its line ("Result:", "Résultat :"), after a lead of
// at most two words ending in a colon ("Done: \"All tests pass.\""), or when the answer is
// little more than the quote itself.
var resultLabel = regexp.MustCompile(`(?i)^(?:[-*>]\s*)?\**(?:result|results|verdict|output|r[ée]sultats?|sortie|statut|status)\**\s*:\s*$`)
var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)
var thirdPerson = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(?:il|elle|ils|elles|l['’]auteur|the author|he|she|they|the user|l['’]utilisateur|le client|the customer|celui-ci|celle-ci|someone|quelqu['’]un)(?:$|[^\p{L}])`)
var firstPerson = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(?:je|j['’]|i|i['’]ve|i['’]m|we|nous|on|my|mon|ma|mes)(?:$|[^\p{L}])`)
var exampleLead = regexp.MustCompile(`(?i)^(?:par exemple|for example|for instance|e\.?g\.?|ex\.?|exemple|example|such as|like|comme|says?|dit|écrit|wrote|writes)\s*:$`)

func stripMentions(text string) string {
	ms := quoted.FindAllStringIndex(text, -1)
	if len(ms) == 0 {
		return text
	}
	rest := text
	for i := len(ms) - 1; i >= 0; i-- {
		rest = rest[:ms[i][0]] + " " + rest[ms[i][1]:]
	}
	if len(wordRe.FindAllString(rest, -1)) < 4 {
		return text // the answer is essentially the quote: treat it as asserted
	}
	var b strings.Builder
	last := 0
	for _, m := range ms {
		b.WriteString(text[last:m[0]])
		start := strings.LastIndexAny(text[:m[0]], "\n.!?") + 1
		lead := strings.TrimSpace(text[start:m[0]])
		asserted := resultLabel.MatchString(lead) ||
			(strings.HasSuffix(lead, ":") && len(wordRe.FindAllString(lead, -1)) <= 2 && !exampleLead.MatchString(lead))
		if asserted {
			b.WriteString(text[m[0]:m[1]])
		} else {
			b.WriteString(" ")
		}
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

// A markdown blockquote ("> ...") reproduces someone else's words, such as an old answer being
// audited. It is a mention, unless the answer is little more than the blockquote itself.
var blockquoteLine = regexp.MustCompile(`(?m)^[ \t]*>.*$`)

func stripBlockquotes(text string) string {
	rest := blockquoteLine.ReplaceAllString(text, " ")
	if rest == text || len(wordRe.FindAllString(rest, -1)) < 4 {
		return text
	}
	return rest
}
