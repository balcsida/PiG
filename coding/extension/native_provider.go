package extension

import (
	"context"
	"encoding/json"

	"github.com/MichaelKinsy/PiG/ai"
)

// NativeProvider is a session-owned carrier for Pi's native Provider callbacks.
// Metadata is immutable; refresh publishes replacement model snapshots. The host
// owns credentials and persistence, and every callback receives its operation context.
type NativeProvider struct {
	// IsCurrent guards publication when host registrations overlap across connections.
	IsCurrent                func() bool
	ID                       string
	Name                     string
	BaseURL                  string
	Models                   []ProviderModelConfig
	CheckAuth                func(context.Context, *ai.Credential) (*ai.AuthCheck, error)
	FilterModels             func(context.Context, []ProviderModelConfig, *ai.Credential) ([]ProviderModelConfig, error)
	ResolveAuth              func(context.Context, *ai.Credential, ai.AuthResolutionOverrides) (*ai.AuthResult, *ai.Credential, error)
	ResolveRefreshCredential func(context.Context, *ai.Credential) (*ai.Credential, *ai.Credential, error)
	RefreshModels            func(context.Context, *ai.Credential, *ai.ModelsStoreEntry, bool, *bool, func(NativeProviderPublication) error) ([]ProviderModelConfig, error)
	Stream                   func(context.Context, *ai.Model, ai.TranscriptContext, ai.StreamOptions, bool) (*ai.AssistantMessageEventStream, error)
	// GenerateImages and Classify are the provider object's image and classifier operations; nil when it has none.
	// upstream: pi-ai Provider.generateImages, Provider.classify
	GenerateImages func(context.Context, *ai.ImageModel, ai.ImagesContext, ai.ImagesOptions) (ai.AssistantImages, error)
	Classify       func(context.Context, *ai.ClassifierModel, ai.ClassifierContext, ai.ClassifierOptions) (ai.ClassifierResult, error)
	// OAuth is the account login method the provider object declares (its auth.oauth). The host publishes the flow
	// itself in the OAuth provider registry; the registry needs only to know the method exists to compose it into the
	// provider's auth, as it does for a configuration registration's oauth.
	OAuth *NativeProviderOAuth
	// APIKey is the API-key login method the provider object declares: its name, and its own login when it has one.
	// Resolution and checks stay behind CheckAuth and ResolveAuth; the registry needs the method to list and run the
	// login, as Pi runs a provider's own apiKey.login.
	APIKey *NativeProviderAPIKey
}

// NativeProviderAPIKey names a provider object's API-key login method.
type NativeProviderAPIKey struct {
	Name  string
	Login func(context.Context, ai.AuthInteraction) (ai.Credential, error)
}

// NativeProviderOAuth names a provider object's account login method.
type NativeProviderOAuth struct {
	Name           string
	IsSubscription bool
	// LoginLabel is the flow's own sign-in label, shown where Pi shows provider.auth.oauth.loginLabel.
	LoginLabel string
}

type NativeProviderPublication struct {
	Persist json.RawMessage        `json:"persist,omitempty"`
	Models  *[]ProviderModelConfig `json:"models,omitempty"`
}

type providerStreamMethodKey struct{}

// WithProviderStreamSimple carries the selected Pi stream method through the host request.
func WithProviderStreamSimple(ctx context.Context, simple bool) context.Context {
	return context.WithValue(ctx, providerStreamMethodKey{}, simple)
}
func ProviderStreamSimpleSelected(ctx context.Context) bool {
	value, _ := ctx.Value(providerStreamMethodKey{}).(bool)
	return value
}
