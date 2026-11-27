package api

import (
	"encoding/json"
	"strings"
)

type playerPart struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// splitPlayerActions recognizes pairs of single asterisks. Runs such as **
// and escaped asterisks stay literal.
func splitPlayerActions(input string) []playerPart {
	var parts []playerPart
	start, open := 0, -1
	for i := 0; i < len(input); {
		if input[i] == '\\' && i+1 < len(input) && (input[i+1] == '\\' || input[i+1] == '*') {
			i += 2
			continue
		}
		if input[i] != '*' {
			i++
			continue
		}
		mark := i
		for i < len(input) && input[i] == '*' {
			i++
		}
		if i-mark != 1 {
			continue
		}
		if open < 0 {
			open = mark
			continue
		}
		if strings.TrimSpace(input[open+1:mark]) == "" {
			open = mark
			continue
		}
		if start < open {
			parts = append(parts, playerPart{"text", input[start:open]})
		}
		parts = append(parts, playerPart{"unspoken_action", input[open+1 : mark]})
		start, open = i, -1
	}
	if len(parts) > 0 && start < len(input) {
		parts = append(parts, playerPart{"text", input[start:]})
	}
	return parts
}

func playerInputForGM(input string) string {
	parts := splitPlayerActions(input)
	if len(parts) == 0 {
		return input
	}
	data, _ := json.Marshal(parts) // Only string fields.
	return string(data)
}
