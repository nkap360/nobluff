package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

// These are bounded command heuristics, not a shell interpreter. In particular, an
// argument containing 'pytest' is not an execution of pytest. Quoted separators
// remain arguments. Custom wrappers and dynamically constructed commands are unknown.
func commandParts(command string) [][]string {
	var parts [][]string
	var words []string
	var word strings.Builder
	quote := rune(0)
	escaped := false
	flushWord := func() {
		if word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	flushPart := func() {
		flushWord()
		if len(words) > 0 {
			parts = append(parts, words)
			words = nil
		}
	}
	for _, c := range command {
		if escaped {
			word.WriteRune(c)
			escaped = false
			continue
		}
		if c == '\\' && quote != '\'' {
			escaped = true
			word.WriteRune(c)
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			} else {
				word.WriteRune(c)
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		switch c {
		case ' ', '\t':
			flushWord()
		case ';', '|', '&', '\n':
			flushPart()
		default:
			word.WriteRune(c)
		}
	}
	flushPart()
	return parts
}

func invocation(words []string) []string {
	for len(words) > 0 && (strings.Contains(words[0], "=") || words[0] == "env" || words[0] == "command") {
		words = words[1:]
	}
	if len(words) > 0 {
		words = append([]string(nil), words...)
		words[0] = strings.ToLower(filepath.Base(strings.ReplaceAll(words[0], "\\", "/")))
		words[0] = strings.TrimSuffix(words[0], ".exe")
	}
	return words
}

func isTestCommand(command string) bool {
	for _, raw := range commandParts(command) {
		w := invocation(raw)
		if len(w) == 0 {
			continue
		}
		// Match the documented runner only at the executable position.
		switch w[0] {
		case "pytest", "py.test", "unittest", "jest", "vitest", "mocha", "rspec", "phpunit", "ctest", "tox", "nox":
			return true
		case "python", "python3", "python3.10", "python3.11", "python3.12", "python3.13", "python3.14", "py":
			w = pythonArgs(w)
			if len(w) > 2 && w[1] == "-m" && (w[2] == "pytest" || w[2] == "unittest") {
				return true
			}
			if len(w) > 1 && !strings.HasPrefix(w[1], "-") && strings.Contains(filepath.Base(w[1]), "test") && strings.HasSuffix(w[1], ".py") {
				return true
			}
		case "npm", "pnpm", "yarn", "bun":
			if len(w) > 1 && w[1] == "test" || len(w) > 2 && w[1] == "run" && w[2] == "test" {
				return true
			}
		case "go", "cargo", "dotnet", "deno", "mvn", "gradle", "gradlew", "./gradlew":
			if len(w) > 1 && w[1] == "test" {
				return true
			}
		case "make":
			if len(w) > 1 && (w[1] == "test" || w[1] == "check") {
				return true
			}
		case "npx":
			if len(w) > 1 && (w[1] == "jest" || w[1] == "vitest" || w[1] == "mocha") {
				return true
			}
		}
	}
	return false
}

func isCheckCommand(command string) bool {
	if isTestCommand(command) {
		return true
	}
	for _, raw := range commandParts(command) {
		w := invocation(raw)
		if len(w) < 2 {
			continue
		}
		switch w[0] {
		case "go", "cargo", "dotnet":
			if w[1] == "build" || w[1] == "check" || w[1] == "vet" {
				return true
			}
		case "python", "python3", "python3.10", "python3.11", "python3.12", "python3.13", "python3.14", "py":
			if w = pythonArgs(w); len(w) < 2 {
				continue
			}
			if w[1] == "-c" || w[1] == "-m" || strings.HasSuffix(w[1], ".py") {
				return true
			}
		case "node":
			if w[1] == "--check" || strings.HasSuffix(w[1], ".js") {
				return true
			}
		case "tsc", "ruff", "mypy", "eslint":
			return true
		}
	}
	return false
}

var shellMutation = regexp.MustCompile(`(?i)(\bsed\s+[^;\n]*-i\b|\b(?:perl|ruby)\s+-[a-z]*i\b|\b(?:tee|touch|truncate|cp|mv|rm|del|copy|move|remove-item|set-content|add-content|out-file)\b|(?:^|\s)(?:>>?|\d?>)\s*[^&\s]|\b(?:write_text|write_bytes|writeFileSync|apply_patch)\b)`)

func shellMayEdit(command string) bool { return shellMutation.MatchString(command) }

// pythonArgs drops interpreter options placed before the script or -m/-c ("python -P -m pytest",
// "python -I x.py", "py -3.12 -m pytest", "python -X utf8 -m unittest"), so they don't hide the runner.
func pythonArgs(w []string) []string {
	out := []string{w[0]}
	i := 1
	for i < len(w) && strings.HasPrefix(w[i], "-") && w[i] != "-m" && w[i] != "-c" && w[i] != "-" {
		if w[i] == "-X" || w[i] == "-W" {
			i++
		}
		i++
	}
	return append(out, w[min(i, len(w)):]...)
}

// shellEditOrder places the file changes of a shell command relative to the checks it runs:
// "none" (no change, or only build artifacts deleted), "before" (every change precedes the last
// test or check of the command, as in `sed -i … && pytest`), or "after" (a change follows the
// last check, or cannot be attributed to a part of the command).
func shellEditOrder(command string) string {
	if !shellMayEdit(command) || cleanupOnly(command) {
		return "none"
	}
	lastCheck, lastMutation := -1, -1
	for i, p := range commandParts(command) {
		joined := strings.Join(p, " ")
		if isCheckCommand(joined) {
			lastCheck = i
		}
		if shellMutation.MatchString(joined) {
			lastMutation = i
		}
	}
	if lastMutation >= 0 && lastCheck >= 0 && lastMutation <= lastCheck {
		return "before"
	}
	return "after"
}

// Build output, caches and screenshots: deleting them cannot change what the tests exercise.
var artifactDir = map[string]bool{".playwright-mcp": true, "node_modules": true, "dist": true, "build": true,
	"__pycache__": true, ".pytest_cache": true, ".mypy_cache": true, ".ruff_cache": true, "coverage": true,
	".cache": true, ".next": true, "tmp": true, ".tmp": true}
var artifactExt = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp|log|tmp)$`)

func isArtifactPath(p string) bool {
	p = strings.Trim(strings.ReplaceAll(strings.TrimRight(p, ","), "\\", "/"), "/")
	if p == "" || p == "." || p == "*" {
		return false
	}
	if proseExt.MatchString(p) || artifactExt.MatchString(p) {
		return true
	}
	for _, seg := range strings.Split(p, "/") {
		if artifactDir[strings.ToLower(seg)] {
			return true
		}
	}
	return false
}

// cleanupOnly is true when every changing part of the command deletes build artifacts, prose or
// screenshots, and nothing else: `rm -rf .playwright-mcp`, `Remove-Item -Recurse node_modules`.
func cleanupOnly(command string) bool {
	deleted := false
	for _, p := range commandParts(command) {
		w := invocation(p)
		if len(w) == 0 || !shellMutation.MatchString(strings.Join(p, " ")) {
			continue
		}
		switch w[0] {
		case "rm", "del", "rmdir", "rd", "remove-item":
		default:
			return false
		}
		targets := 0
		for _, a := range w[1:] {
			if strings.HasPrefix(a, "-") || a == "," {
				continue
			}
			if !isArtifactPath(a) {
				return false
			}
			targets++
		}
		if targets == 0 {
			return false
		}
		deleted = true
	}
	return deleted
}
