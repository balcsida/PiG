package codingagent

import (
	"context"
	"testing"

	"github.com/MichaelKinsy/PiG/ai"
	"github.com/MichaelKinsy/PiG/coding/extension"
)

// A provider object whose auth declares an OAuth method is an account login (Pi composes the object's own
// auth.oauth, model-runtime.ts registerNativeProvider). The host publishes the flow in the OAuth provider registry;
// the registry must compose it into the provider's auth, as it does for a configuration registration's oauth,
// or /login never offers "Sign in with an account" for the provider.
func TestNativeProviderOAuthMethodComposesIntoLoginAuth(t *testing.T) {
	const id = "external-oauth-test" // the id testExternalOAuthProvider declares
	ai.RegisterOAuthProvider(id, testExternalOAuthProvider{})
	defer ai.UnregisterOAuthProvider(id)
	makeProvider := func(oauth *extension.NativeProviderOAuth) *extension.NativeProvider {
		return &extension.NativeProvider{ID: id, Name: "External", OAuth: oauth,
			Models: []extension.ProviderModelConfig{{ID: "model", API: ai.APIOpenAICompletions, BaseURL: "https://native.invalid"}},
			CheckAuth: func(context.Context, *ai.Credential) (*ai.AuthCheck, error) {
				return &ai.AuthCheck{Type: ai.CredentialAPIKey}, nil
			},
			ResolveAuth: func(context.Context, *ai.Credential, ai.AuthResolutionOverrides) (*ai.AuthResult, *ai.Credential, error) {
				return &ai.AuthResult{Auth: ai.ModelAuth{APIKey: "key"}}, nil, nil
			},
			ResolveRefreshCredential: func(context.Context, *ai.Credential) (*ai.Credential, *ai.Credential, error) { return nil, nil, nil },
			Stream: func(context.Context, *ai.Model, ai.TranscriptContext, ai.StreamOptions, bool) (*ai.AssistantMessageEventStream, error) {
				return ai.NewAssistantMessageEventStream(), nil
			},
		}
	}

	t.Run("a declared method is inherited from the published flow", func(t *testing.T) {
		registry := NewModelRegistry(t.TempDir())
		if err := registry.RegisterNativeProvider(t.Context(), makeProvider(&extension.NativeProviderOAuth{Name: "External SSO", LoginLabel: "Sign in with External"})); err != nil {
			t.Fatal(err)
		}
		auth, config, ok := registry.registryAuthConfig(id)
		if !ok {
			t.Fatal("registryAuthConfig does not know the provider")
		}
		if config.OAuth == nil || !config.OAuth.HasLogin || config.OAuth.Name != "External SSO" {
			t.Fatalf("registration does not record the OAuth method: %+v", config.OAuth)
		}
		if auth.OAuth == nil || auth.OAuth.Login == nil {
			t.Fatal("composed auth has no OAuth login, so /login would offer no account sign-in")
		}
		if auth.OAuth.LoginLabel != "Sign in with External" {
			t.Fatalf("the flow's own login label is not shown: %q", auth.OAuth.LoginLabel)
		}
		// The refresh that follows every registration rewrites the registration from the provider's catalog; the
		// method must survive it, or the account login disappears seconds after /login first offered it.
		provider := registry.NativeProvider(id)
		if err := registry.refreshNativeProvider(t.Context(), provider, false, nil); err != nil {
			t.Fatal(err)
		}
		if auth, _, _ := registry.registryAuthConfig(id); auth.OAuth == nil || auth.OAuth.Login == nil {
			t.Fatal("the catalog refresh dropped the OAuth method from the registration")
		}
	})

	t.Run("the object's API-key method survives beside its OAuth method", func(t *testing.T) {
		registry := NewModelRegistry(t.TempDir())
		provider := makeProvider(&extension.NativeProviderOAuth{Name: "External SSO", LoginLabel: "Sign in with External"})
		var loginCalled bool
		provider.APIKey = &extension.NativeProviderAPIKey{Name: "External API key", Login: func(context.Context, ai.AuthInteraction) (ai.Credential, error) {
			loginCalled = true
			return ai.Credential{Type: ai.CredentialAPIKey, Key: "k"}, nil
		}}
		if err := registry.RegisterNativeProvider(t.Context(), provider); err != nil {
			t.Fatal(err)
		}
		auth, _, _ := registry.registryAuthConfig(id)
		if auth.OAuth == nil || auth.APIKey == nil {
			t.Fatalf("both methods must be composed: oauth=%v apiKey=%v", auth.OAuth != nil, auth.APIKey != nil)
		}
		if auth.APIKey.Name != "External API key" {
			t.Fatalf("the API-key method lost its declared name: %q", auth.APIKey.Name)
		}
		if _, err := auth.APIKey.Login(t.Context(), ai.AuthInteraction{}); err != nil || !loginCalled {
			t.Fatalf("the object's own login is not run: %v (called=%v)", err, loginCalled)
		}
	})

	t.Run("a provider object without an OAuth method composes none", func(t *testing.T) {
		registry := NewModelRegistry(t.TempDir())
		if err := registry.RegisterNativeProvider(t.Context(), makeProvider(nil)); err != nil {
			t.Fatal(err)
		}
		auth, _, ok := registry.registryAuthConfig(id)
		if !ok {
			t.Fatal("registryAuthConfig does not know the provider")
		}
		if auth.OAuth != nil {
			t.Fatal("composed auth offers an OAuth login the provider object never declared")
		}
	})
}
