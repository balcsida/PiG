package subprocess

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/MichaelKinsy/PiG/ai"
)

// The extension's API-key login sends the same prompt and notify requests as its OAuth login; the host answers them
// with the dialog's interaction and returns the chosen value, so a flow can confirm a known proxy URL and then read
// a secret, as Pi's showApiKeyLoginDialog lets a provider's own apiKey.login do.
func TestNativeAuthInteractionAnswersPromptsAndEvents(t *testing.T) {
	var prompts []ai.AuthPrompt
	var events []ai.AuthEvent
	interaction := ai.AuthInteraction{
		Prompt: func(_ context.Context, prompt ai.AuthPrompt) (string, error) {
			prompts = append(prompts, prompt)
			switch typed := prompt.(type) {
			case ai.AuthSelectPrompt:
				return typed.Options[1].ID, nil
			case ai.AuthSecretPrompt:
				return "sk-secret", nil
			}
			return "", errors.New("unexpected prompt")
		},
		Notify: func(event ai.AuthEvent) { events = append(events, event) },
	}
	callback := nativeAuthInteraction(context.Background(), interaction)

	answer, err := callback(json.RawMessage(`{"method":"prompt","params":{"prompt":{"type":"select","message":"Proxy URL:","options":[{"id":"a","label":"A"},{"id":"b","label":"B (known)"}]}}}`))
	if err != nil || string(answer) != `"b"` {
		t.Fatalf("select prompt answered %s, %v", answer, err)
	}
	answer, err = callback(json.RawMessage(`{"method":"prompt","params":{"prompt":{"type":"secret","message":"Enter API key:"}}}`))
	if err != nil || string(answer) != `"sk-secret"` {
		t.Fatalf("secret prompt answered %s, %v", answer, err)
	}
	answer, err = callback(json.RawMessage(`{"method":"notify","params":{"event":{"type":"device_code","userCode":"ABCD","verificationUri":"https://proxy.example/sso","expiresInSeconds":600}}}`))
	if err != nil || string(answer) != "null" {
		t.Fatalf("notify answered %s, %v", answer, err)
	}
	if len(prompts) != 2 || len(events) != 1 {
		t.Fatalf("saw %d prompts and %d events", len(prompts), len(events))
	}
	if secret, ok := prompts[1].(ai.AuthSecretPrompt); !ok || secret.Message != "Enter API key:" {
		t.Fatalf("second prompt is %#v", prompts[1])
	}
	if code, ok := events[0].(ai.AuthDeviceCodeEvent); !ok || code.UserCode != "ABCD" || code.ExpiresInSeconds == nil || *code.ExpiresInSeconds != 600 {
		t.Fatalf("event is %#v", events[0])
	}
}

func TestNativeAuthInteractionReportsCancelAndUnknownShapes(t *testing.T) {
	cancelled := errors.New("Login cancelled")
	callback := nativeAuthInteraction(context.Background(), ai.AuthInteraction{
		Prompt: func(context.Context, ai.AuthPrompt) (string, error) { return "", cancelled },
	})
	if _, err := callback(json.RawMessage(`{"method":"prompt","params":{"prompt":{"type":"text","message":"URL"}}}`)); !errors.Is(err, cancelled) {
		t.Fatalf("a cancelled prompt returned %v", err)
	}
	if _, err := callback(json.RawMessage(`{"method":"prompt","params":{"prompt":{"type":"picture"}}}`)); err == nil {
		t.Fatal("an unknown prompt type was accepted")
	}
	if _, err := callback(json.RawMessage(`{"method":"dance","params":{}}`)); err == nil {
		t.Fatal("an unknown callback method was accepted")
	}
}

// A declared auth.apiKey.login method is carried to the registry; without it the carrier has no login, and Pi's
// generic API-key prompt applies.
func TestNativeProviderAuthCarriesDeclaredAPIKeyLogin(t *testing.T) {
	declared := &nativeProviderProxy{declaration: NativeProviderDeclaration{ID: "p", Auth: &ProviderObjectAuthDeclaration{APIKey: &ProviderObjectAuthMethodDeclaration{Name: "key"}}, Methods: []string{"auth.apiKey.resolve", "auth.apiKey.login"}}}
	if declared.auth(context.Background()).APIKey.Login == nil {
		t.Fatal("a declared apiKey.login is not carried")
	}
	undeclared := &nativeProviderProxy{declaration: NativeProviderDeclaration{ID: "p", Auth: &ProviderObjectAuthDeclaration{APIKey: &ProviderObjectAuthMethodDeclaration{Name: "key"}}, Methods: []string{"auth.apiKey.resolve"}}}
	if undeclared.auth(context.Background()).APIKey.Login != nil {
		t.Fatal("an undeclared apiKey.login is invented")
	}
}
