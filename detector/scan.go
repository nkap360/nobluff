package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Everything below runs locally: transcripts are read from disk and nothing is sent anywhere.

func projectsDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "projects")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

type sessionFile struct {
	path    string
	project string
	mod     time.Time
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// sessions lists top-level session files (subagent transcripts live in subfolders), newest first.
func sessions(root, onlyProject string) []sessionFile {
	var out []sessionFile
	projects, _ := os.ReadDir(root)
	for _, p := range projects {
		if !p.IsDir() || (onlyProject != "" && p.Name() != onlyProject) {
			continue
		}
		files, _ := os.ReadDir(filepath.Join(root, p.Name()))
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			out = append(out, sessionFile{filepath.Join(root, p.Name(), f.Name()), p.Name(), info.ModTime()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].mod.After(out[j].mod) })
	return out
}

type hit struct {
	f       Finding
	project string
	when    time.Time
}

func scanCmd(args []string, w io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	limit := fs.Int("n", 50, "number of most recent sessions to scan")
	dir := fs.String("dir", projectsDir(), "Claude Code projects folder")
	if fs.Parse(args) != nil {
		return 2
	}
	files := sessions(*dir, "")
	if len(files) == 0 {
		fmt.Fprintf(w, "No Claude Code sessions found in %s\n", *dir)
		return 0
	}
	if len(files) > *limit {
		files = files[:*limit]
	}
	return report(w, files)
}

// lastCmd checks the most recent session of the current folder (or of any folder if none).
func lastCmd(w io.Writer) int {
	cwd, _ := os.Getwd()
	files := sessions(projectsDir(), nonAlnum.ReplaceAllString(cwd, "-"))
	if len(files) == 0 {
		files = sessions(projectsDir(), "")
	}
	if len(files) == 0 {
		fmt.Fprintf(w, "No Claude Code sessions found in %s\n", projectsDir())
		return 0
	}
	return report(w, files[:1])
}

func report(w io.Writer, files []sessionFile) int {
	var hits []hit
	turns, answered, flaggedAnswers, unreadable := 0, 0, 0, 0
	rules := map[string]int{}
	flagged := map[string]bool{}
	for _, sf := range files {
		ts, err := Turns(sf.path)
		if err != nil {
			unreadable++
			continue
		}
		for _, t := range ts {
			turns++
			if strings.TrimSpace(t.FinalText) != "" {
				answered++
			}
			findings := Check(t)
			if len(findings) > 0 {
				flaggedAnswers++
			}
			for _, f := range findings {
				rules[f.Rule]++
				hits = append(hits, hit{f, sf.project, sf.mod})
				flagged[sf.path] = true
			}
		}
	}
	plural := func(n int, s string) string {
		if n == 1 {
			return fmt.Sprintf("%d %s", n, s)
		}
		return fmt.Sprintf("%d %ss", n, s)
	}
	fmt.Fprintf(w, "NoBluff scanned %s (%s, %s).\n", plural(len(files), "Claude Code session"), plural(turns, "turn"), plural(answered, "final answer"))
	if unreadable > 0 {
		fmt.Fprintf(w, "Incomplete coverage: %d session(s) could not be parsed.\n", unreadable)
	}
	if len(hits) == 0 {
		fmt.Fprintln(w, "No answer claimed something its own transcript contradicts.")
		if unreadable > 0 {
			return 2
		}
		return 0
	}
	fmt.Fprintf(w, "%s had %s to review, in %s.\n\n", plural(flaggedAnswers, "answer"), plural(len(hits), "finding"), plural(len(flagged), "session"))
	names := make([]string, 0, len(rules))
	for r := range rules {
		names = append(names, r)
	}
	sort.Slice(names, func(i, j int) bool { return rules[names[i]] > rules[names[j]] })
	for _, r := range names {
		fmt.Fprintf(w, "  %-18s %d\n", r, rules[r])
	}
	fmt.Fprintln(w, "\nExamples:")
	for i, h := range hits {
		if i == 5 {
			break
		}
		claim := h.f.Claim
		if len(claim) > 110 {
			claim = claim[:107] + "..."
		}
		fmt.Fprintf(w, "  [%s] %s  %q\n", h.when.Format("2006-01-02"), h.f.Rule, claim)
	}
	fmt.Fprintln(w, "\nNothing left your machine. Block these at the source: /plugin install nobluff@provenwork")
	return 1
}
