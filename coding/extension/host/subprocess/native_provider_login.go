package subprocess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MichaelKinsy/PiG/ai"
)

// nativeAuthInteraction answers a provider object's login callbacks with the host's ai.AuthInteraction, so the
// extension's own API-key login drives the same dialog prompts and events as a built-in provider's. The extension
// sends `prompt` and `notify` requests shaped as the OAuth login's (see nativeOAuthProxy.LoginContext); a prompt's
// answer is its string value, and a notification is acknowledged with null.
func nativeAuthInteraction(ctx context.Context, interaction ai.AuthInteraction) func(json.RawMessage) (json.RawMessage, error) {
	return func(data json.RawMessage) (json.RawMessage, error) {
		var request struct {
			Method string `json:"method"`
			Params struct {
				Prompt json.RawMessage `json:"prompt"`
				Event  json.RawMessage `json:"event"`
			} `json:"params"`
		}
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		switch request.Method {
		case "prompt":
			if interaction.Prompt == nil {
				return nil, errors.New("login prompts are not available")
			}
			prompt, err := decodeAuthPrompt(request.Params.Prompt)
			if err != nil {
				return nil, err
			}
			value, err := interaction.Prompt(ctx, prompt)
			if err != nil {
				return nil, err
			}
			return json.Marshal(value)
		case "notify":
			event, err := decodeAuthEvent(request.Params.Event)
			if err != nil {
				return nil, err
			}
			if interaction.Notify != nil {
				interaction.Notify(event)
			}
			return json.RawMessage("null"), nil
		default:
			return nil, fmt.Errorf("unexpected login callback %s", request.Method)
		}
	}
}

// decodeAuthPrompt reads the wire form of Pi's AuthPrompt union, discriminated by type.
func decodeAuthPrompt(data json.RawMessage) (ai.AuthPrompt, error) {
	var kind struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &kind); err != nil {
		return nil, err
	}
	switch kind.Type {
	case "text":
		var prompt ai.AuthTextPrompt
		return prompt, json.Unmarshal(data, &prompt)
	case "secret":
		var prompt ai.AuthSecretPrompt
		return prompt, json.Unmarshal(data, &prompt)
	case "manual_code":
		var prompt ai.AuthManualCodePrompt
		return prompt, json.Unmarshal(data, &prompt)
	case "select":
		var prompt ai.AuthSelectPrompt
		return prompt, json.Unmarshal(data, &prompt)
	default:
		return nil, fmt.Errorf("unknown login prompt type %q", kind.Type)
	}
}

// decodeAuthEvent reads the wire form of Pi's AuthEvent union, discriminated by type.
func decodeAuthEvent(data json.RawMessage) (ai.AuthEvent, error) {
	var kind struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &kind); err != nil {
		return nil, err
	}
	switch kind.Type {
	case "info":
		var event ai.AuthInfoEvent
		return event, json.Unmarshal(data, &event)
	case "auth_url":
		var event ai.AuthURLEvent
		return event, json.Unmarshal(data, &event)
	case "device_code":
		var event ai.AuthDeviceCodeEvent
		return event, json.Unmarshal(data, &event)
	case "progress":
		var event ai.AuthProgressEvent
		return event, json.Unmarshal(data, &event)
	default:
		return nil, fmt.Errorf("unknown login event type %q", kind.Type)
	}
}
