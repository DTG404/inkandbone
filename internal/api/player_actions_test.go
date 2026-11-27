package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitPlayerActions(t *testing.T) {
	for _, example := range []struct {
		input string
		want  []playerPart
	}{
		{"*looks around confused*", []playerPart{{"unspoken_action", "looks around confused"}}},
		{"*转身* “Who’s there?” *waits*!", []playerPart{{"unspoken_action", "转身"}, {"text", " “Who’s there?” "}, {"unspoken_action", "waits"}, {"text", "!"}}},
		{"*looks\naround*", []playerPart{{"unspoken_action", "looks\naround"}}},
		{"I look around.", nil},
		{"**bold** ***bold italic*** *unclosed", nil},
		{"*  *", nil},
		{`\*literal\* and *nods*`, []playerPart{{"text", `\*literal\* and `}, {"unspoken_action", "nods"}}},
	} {
		assert.Equal(t, example.want, splitPlayerActions(example.input), example.input)
		if example.want == nil {
			assert.Equal(t, example.input, playerInputForGM(example.input))
		}
	}
}
