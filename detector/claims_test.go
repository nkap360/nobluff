package main

import "testing"

// Field report 2026-11-09, ticket 1: questions, offers, hypothetical subordinates and two-word
// French negations assert nothing. Each removed false alarm is paired with a near-identical
// sentence that is a real claim and must stay flagged.
func TestQuestionsOffersSubordinatesNegations(t *testing.T) {
	cases := []struct {
		name, say, want string
	}{
		// offers
		{"fr_offer", "Je peux ensuite chercher une sortie récente vérifiée à laquelle l'arrimer.", ""},
		{"en_offer", "I can verify the fix on Windows next.", ""},
		{"en_offer_to_confirm_is_a_claim", "I can confirm the tests pass.", "tests-not-run"},
		{"fr_offer_to_confirm_is_a_claim", "Je peux confirmer que les tests passent.", "tests-not-run"},
		// questions
		{"fr_question", "Tu veux que je vérifie le masquage des secrets ?", ""},
		{"en_question", "Want me to check that the tests pass?", ""},
		{"question_then_claim", "Tests pass? Yes: all tests pass.", "tests-not-run"},
		// hypothetical subordinates
		{"fr_until", "Je fixe la liste avant de voir le résultat, pour ne pas ajuster les règles jusqu'à ce que le test passe.", ""},
		{"en_until", "Keep the flag on until all tests pass.", ""},
		{"fr_singular_claim", "Le test passe.", "tests-not-run"},
		// two-word French negations
		{"fr_neg_two_words", "Le rendu mobile n'a pas été vérifié en capture.", ""},
		{"fr_passive_claim", "Le rendu mobile a été vérifié en capture.", "unverified-claim"},
		{"fr_first_person_claim_stays", "J'ai vérifié, ça fonctionne.", "unverified-claim"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { expect(t, (&tb{}).edit("app.py").say(c.say).check(t), c.want) })
	}
}
