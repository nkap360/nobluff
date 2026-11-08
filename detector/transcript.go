package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
)

// Step is one thing the agent actually did, in order.
type Step struct {
	Kind      string // "edit" or "command"
	Tool      string // Edit, Write, MultiEdit, NotebookEdit, Bash, PowerShell
	Command   string // shell command for Kind == "command"
	Path      string // edited file for Kind == "edit"
	Output    string // tool result text
	Failed    bool   // tool result flagged as error
	HasResult bool   // a completed tool result was observed
	ID        string
}

// Session is what the detector knows: the steps, and what the agent said at the end.
type Session struct {
	Steps     []Step
	FinalText string
}

type event struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

var editTools = map[string]bool{"Edit": true, "Write": true, "MultiEdit": true, "NotebookEdit": true}
var shellTools = map[string]bool{"Bash": true, "PowerShell": true}

// LoadSession reads a Claude Code transcript: JSON Lines (the session file) or a JSON array (BluffBench).
func LoadSession(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseSession(f)
}

func ParseSession(r io.Reader) (*Session, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if ma := managedEvents(data); ma != nil {
		// Managed Agents: judge the latest turn, like the Stop hook judges the answer being given now.
		turns := managedTurns(ma)
		if len(turns) == 0 {
			return &Session{}, nil
		}
		return turns[len(turns)-1], nil
	}
	events, err := readEvents(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return build(events), nil
}

// Turns splits a transcript at each new user prompt and returns one Session per turn, so a
// claim in turn 3 is judged against what ran in turn 3. Hook feedback also starts a turn.
func Turns(path string) ([]*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if ma := managedEvents(data); ma != nil {
		return managedTurns(ma), nil
	}
	events, err := readEvents(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var out []*Session
	start := 0
	for i, e := range events {
		if i > start && isPrompt(e) {
			out = append(out, build(events[start:i]))
			start = i
		}
	}
	if start < len(events) {
		out = append(out, build(events[start:]))
	}
	return out, nil
}

func isPrompt(e event) bool {
	if e.Type != "user" {
		return false
	}
	var str string
	if json.Unmarshal(e.Message.Content, &str) == nil {
		return true
	}
	var blocks []block
	if json.Unmarshal(e.Message.Content, &blocks) != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "text" {
			return true
		}
	}
	return false
}

func readEvents(r io.Reader) ([]event, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var events []event
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &events); err != nil {
			return nil, err
		}
	} else {
		sc := bufio.NewScanner(bytes.NewReader(data))
		sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
		for sc.Scan() {
			var e event
			if len(bytes.TrimSpace(sc.Bytes())) == 0 {
				continue
			}
			if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
				return nil, err
			}
			events = append(events, e)
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func build(events []event) *Session {
	s := &Session{}
	index := map[string]int{}
	var finalParts []string
	for _, e := range events {
		if e.Type != "assistant" && e.Type != "user" {
			continue
		}
		var blocks []block
		if json.Unmarshal(e.Message.Content, &blocks) != nil {
			// a plain-string user message (new prompt, or hook feedback after a block)
			// supersedes the previous answer: only what the agent says after it is final
			if e.Type == "user" {
				finalParts = nil
			}
			continue
		}
		if e.Type == "user" {
			for _, b := range blocks {
				if b.Type == "text" {
					finalParts = nil
					break
				}
			}
		}
		hadTool := false
		var texts []string
		for _, b := range blocks {
			switch b.Type {
			case "tool_use":
				hadTool = true
				var in struct {
					Command      string `json:"command"`
					FilePath     string `json:"file_path"`
					NotebookPath string `json:"notebook_path"`
				}
				_ = json.Unmarshal(b.Input, &in)
				switch {
				case editTools[b.Name]:
					p := in.FilePath
					if p == "" {
						p = in.NotebookPath
					}
					s.Steps = append(s.Steps, Step{Kind: "edit", Tool: b.Name, Path: p, ID: b.ID})
				case shellTools[b.Name]:
					s.Steps = append(s.Steps, Step{Kind: "command", Tool: b.Name, Command: in.Command, ID: b.ID})
				default:
					continue
				}
				index[b.ID] = len(s.Steps) - 1
				if shellTools[b.Name] && shellMayEdit(in.Command) {
					s.Steps = append(s.Steps, Step{Kind: "edit", Tool: b.Name})
				}
			case "tool_result":
				if i, ok := index[b.ToolUseID]; ok {
					s.Steps[i].Output = resultText(b.Content)
					s.Steps[i].Failed = b.IsError
					s.Steps[i].HasResult = true
				}
			case "text":
				if e.Type == "assistant" {
					texts = append(texts, b.Text)
				}
			}
		}
		if e.Type == "assistant" {
			if hadTool {
				finalParts = nil // text before a tool call is narration, not the final answer
			}
			if len(texts) > 0 {
				finalParts = append(finalParts, texts...)
			}
		}
	}
	s.FinalText = strings.Join(finalParts, "\n")
	return s
}

func resultText(raw json.RawMessage) string {
	var str string
	if json.Unmarshal(raw, &str) == nil {
		return str
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
			b.WriteString("\n")
		}
		return b.String()
	}
	return string(raw)
}
