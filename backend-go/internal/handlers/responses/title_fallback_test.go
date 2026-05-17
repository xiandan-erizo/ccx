package responses

import "testing"

func TestExtractLastResponsesUserInputUsesLastInputText(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{"type": "input_text", "text": "<system-reminder>ignore</system-reminder>"},
				map[string]interface{}{"type": "input_text", "text": "ls"},
			},
		},
		map[string]interface{}{"type": "function_call", "name": "Bash"},
		map[string]interface{}{"type": "function_call_output", "output": "ignored"},
	}

	if got := extractLastResponsesUserInput(input); got != "ls" {
		t.Fatalf("extractLastResponsesUserInput() = %q, want ls", got)
	}
	if got := countResponsesUserMessages(input); got != 1 {
		t.Fatalf("countResponsesUserMessages() = %d, want 1", got)
	}
}

func TestExtractLastResponsesUserInputJoinsShortInputs(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"role": "user", "content": []interface{}{map[string]interface{}{"type": "input_text", "text": "第一个"}}},
		map[string]interface{}{"role": "assistant", "content": []interface{}{map[string]interface{}{"type": "output_text", "text": "回答"}}},
		map[string]interface{}{"role": "user", "content": []interface{}{map[string]interface{}{"type": "input_text", "text": "第二个"}}},
	}

	if got := extractLastResponsesUserInput(input); got != "第一个 / 第二个" {
		t.Fatalf("extractLastResponsesUserInput() = %q, want %q", got, "第一个 / 第二个")
	}
	if got := countResponsesUserMessages(input); got != 2 {
		t.Fatalf("countResponsesUserMessages() = %d, want 2", got)
	}
}
