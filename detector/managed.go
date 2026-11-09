package main

// Claude Managed Agents session events (GET /v1/sessions/{id}/events), mapped onto the same Steps
// as Claude Code transcripts, so every rule applies unchanged.
//
// Documented shapes used here:
//   agent.tool_use    {id, name: "bash"|"write"|"edit"|..., input: {command | file_path ...}}
//   agent.tool_result {content, is_error}   (the field linking it to its tool_use is not documented;
//                                            we accept the usual names, else the latest pending call)
//   agent.message     {content: [{type: "text", text}]}
//   user.message / system.message start a new turn; session.status_idle ends one.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

type maEvent struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Input   json.RawMessage `json:"input"`
	Content json.RawMessage `json:"content"`
	IsError bool            `json:"is_error"`
	// candidate link fields from a tool_result to its tool_use
	ToolUseID      string          `json:"tool_use_id"`
	ToolUseEventID string          `json:"tool_use_event_id"`
	AgentToolUseID string          `json:"agent_tool_use_id"`
	Message        json.RawMessage `json:"message"` // present only in Claude Code transcripts
}

var maEdit = map[string]bool{"write": true, "edit": true}
var maShell = map[string]bool{"bash": true}

// managedEvents returns the events if data is a Managed Agents event list, else nil.
func managedEvents(data []byte) []maEvent {
	trimmed := bytes.TrimSpace(data)
	var events []maEvent
	switch {
	case len(trimmed) > 0 && trimmed[0] == '{' && bytes.Contains(trimmed[:min(len(trimmed), 64)], []byte(`"data"`)):
		var page struct {
			Data []maEvent `json:"data"`
		}
		if json.Unmarshal(trimmed, &page) == nil {
			events = page.Data
		}
	case len(trimmed) > 0 && trimmed[0] == '[':
		_ = json.Unmarshal(trimmed, &events)
	default:
		sc := bufio.NewScanner(bytes.NewReader(trimmed))
		sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
		for sc.Scan() {
			var e maEvent
			if len(bytes.TrimSpace(sc.Bytes())) == 0 {
				continue
			}
			if json.Unmarshal(sc.Bytes(), &e) != nil {
				return nil
			}
			events = append(events, e)
		}
		if sc.Err() != nil {
			return nil
		}
	}
	managed := false
	for _, e := range events {
		if len(e.Message) > 0 {
			return nil // a Claude Code transcript line
		}
		if strings.HasPrefix(e.Type, "agent.") || e.Type == "user.message" || strings.HasPrefix(e.Type, "session.status") {
			managed = true
		}
	}
	if !managed {
		return nil
	}
	return events
}

func textOf(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return resultText(raw)
}

// managedTurns builds one Session per turn (a turn starts at each user.message / system.message).
func managedTurns(events []maEvent) []*Session {
	var out []*Session
	cur := &Session{}
	var finalParts []string
	index := map[string]int{}
	var pending []int // tool_use steps still waiting for their result, oldest first
	flush := func() {
		cur.FinalText = strings.Join(finalParts, "\n")
		if len(cur.Steps) > 0 || cur.FinalText != "" {
			out = append(out, cur)
		}
		cur, finalParts, index, pending = &Session{}, nil, map[string]int{}, nil
	}
	for _, e := range events {
		switch e.Type {
		case "user.message", "system.message":
			flush()
		case "agent.tool_use":
			finalParts = nil // text before a tool call is narration, not the final answer
			var in struct {
				Command  string `json:"command"`
				FilePath string `json:"file_path"`
				Path     string `json:"path"`
			}
			_ = json.Unmarshal(e.Input, &in)
			switch {
			case maShell[e.Name]:
				if shellEditOrder(in.Command) == "before" { // the command changed files before checking them
					cur.Steps = append(cur.Steps, Step{Kind: "edit", Tool: e.Name})
				}
				cur.Steps = append(cur.Steps, Step{Kind: "command", Tool: e.Name, Command: in.Command, ID: e.ID})
			case maEdit[e.Name]:
				p := in.FilePath
				if p == "" {
					p = in.Path
				}
				cur.Steps = append(cur.Steps, Step{Kind: "edit", Tool: e.Name, Path: p, ID: e.ID})
			default:
				continue
			}
			index[e.ID] = len(cur.Steps) - 1
			pending = append(pending, len(cur.Steps)-1)
			if maShell[e.Name] && shellEditOrder(in.Command) == "after" {
				cur.Steps = append(cur.Steps, Step{Kind: "edit", Tool: e.Name})
			}
		case "agent.tool_result":
			i, ok := -1, false
			for _, link := range []string{e.ToolUseID, e.ToolUseEventID, e.AgentToolUseID} {
				if link != "" {
					i, ok = index[link]
					break
				}
			}
			if !ok && len(pending) > 0 {
				i, ok = pending[len(pending)-1], true
			}
			if ok {
				cur.Steps[i].Output = textOf(e.Content)
				cur.Steps[i].Failed = e.IsError
				cur.Steps[i].HasResult = true
				for k, p := range pending {
					if p == i {
						pending = append(pending[:k], pending[k+1:]...)
						break
					}
				}
			}
		case "agent.message":
			if t := textOf(e.Content); strings.TrimSpace(t) != "" {
				finalParts = append(finalParts, t)
			}
		}
	}
	flush()
	return out
}
