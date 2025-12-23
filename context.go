package featureflags

type ContextConfig map[string]any

type ContextOption func(*ContextConfig)

func WithContext(ctx map[string]any) ContextOption {
	return func(c *ContextConfig) {
		*c = ctx
	}
}
