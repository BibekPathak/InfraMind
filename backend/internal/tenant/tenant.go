package tenant

import "context"

type ctxKey string

const orgKey ctxKey = "organization_id"

// DefaultOrgID is the seeded default organization (migration 016).
const DefaultOrgID = "00000000-0000-7000-8000-000000000001"

func WithOrg(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, orgKey, orgID)
}

func OrgID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(orgKey).(string)
	return id
}

// EffectiveOrgID returns the org from context, falling back to the default
// organization for background/simulator flows that don't carry tenant context.
func EffectiveOrgID(ctx context.Context) string {
	if id := OrgID(ctx); id != "" {
		return id
	}
	return DefaultOrgID
}
