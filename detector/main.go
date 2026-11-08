// nobluff: a Claude Code Stop hook that refuses to let the agent finish on a claim its own
// transcript contradicts ("tests pass" with no test run, "fixed" with a failing last run, ...).
//
//	nobluff                     hook mode: reads the Stop hook JSON on stdin
//	nobluff check <transcript>  prints findings as JSON (JSON Lines session file or JSON array)
//	nobluff scan [-n 50]        audits your recent Claude Code sessions, turn by turn, locally
//	nobluff last                audits the latest session of the current folder
//	NOBLUFF_MODE=warn           report to the user instead of blocking
//
// No model, no network: the same transcript always gives the same verdict, in milliseconds.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

var version = "0.4.1"

type hookInput struct {
	TranscriptPath string `json:"transcript_path"`
	StopHookActive bool   `json:"stop_hook_active"`
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "demo":
			// Complete synthetic example, without reading any private session.
			f := Check(&Session{Steps: []Step{{Kind: "edit", Path: "synthetic-example.py"}}, FinalText: "All tests pass."})
			_ = json.NewEncoder(os.Stdout).Encode(f)
			if len(f) > 0 {
				os.Exit(1)
			}
			return
		case "check":
			os.Exit(checkCmd(os.Args[2:]))
		case "scan":
			os.Exit(scanCmd(os.Args[2:], os.Stdout))
		case "last":
			os.Exit(lastCmd(os.Stdout))
		case "version", "--version":
			fmt.Println("nobluff", version)
			return
		}
	}
	os.Exit(hook(os.Stdin, os.Stdout, os.Getenv("NOBLUFF_MODE")))
}

// debugf appends to the file named by NOBLUFF_DEBUG, if set. Silent otherwise.
func debugf(format string, a ...any) {
	path := os.Getenv("NOBLUFF_DEBUG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", a...)
}

func checkCmd(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: nobluff check <transcript>")
		return 2
	}
	var s *Session
	var err error
	if args[0] == "-" { // e.g. piped from GET /v1/sessions/{id}/events
		s, err = ParseSession(os.Stdin)
	} else {
		s, err = LoadSession(args[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "nobluff:", err)
		return 2
	}
	f := Check(s)
	if f == nil {
		f = []Finding{}
	}
	_ = json.NewEncoder(os.Stdout).Encode(f)
	if len(f) > 0 {
		return 1
	}
	return 0
}

// hook never fails the agent's session: any internal problem lets the stop go through.
func hook(stdin io.Reader, stdout io.Writer, mode string) int {
	var in hookInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil || in.TranscriptPath == "" {
		debugf("no usable hook input: %v", err)
		_ = json.NewEncoder(stdout).Encode(map[string]string{"systemMessage": "NoBluff could not evaluate the turn: invalid hook input. This is an error, not a clean verdict."})
		return 0
	}
	s, err := LoadSession(in.TranscriptPath)
	if err != nil {
		debugf("cannot read transcript %q: %v", in.TranscriptPath, err)
		_ = json.NewEncoder(stdout).Encode(map[string]string{"systemMessage": "NoBluff could not evaluate the turn: transcript read/parse error. This is an error, not a clean verdict."})
		return 0
	}
	findings := Check(s)
	debugf("transcript %q: %d steps, final answer %d chars, %d findings", in.TranscriptPath, len(s.Steps), len(s.FinalText), len(findings))
	if len(findings) == 0 {
		return 0
	}
	var reasons []string
	for _, f := range findings {
		reasons = append(reasons, "- "+f.Reason)
	}
	msg := "NoBluff: your final answer claims something this session's transcript contradicts.\n" + strings.Join(reasons, "\n")
	// One correction round only: if we already blocked once, warn instead of looping forever.
	if mode == "warn" || in.StopHookActive {
		_ = json.NewEncoder(stdout).Encode(map[string]string{"systemMessage": msg})
		return 0
	}
	_ = json.NewEncoder(stdout).Encode(map[string]string{"decision": "block", "reason": msg})
	return 0
}
