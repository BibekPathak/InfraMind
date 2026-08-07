package action

import (
	"context"
	"log/slog"

	"github.com/inframind/backend/internal/asset"
)

const (
	modeManual     = "manual"
	modeAdvisory   = "advisory"
	modeAutonomous = "autonomous"
)

// Safe action types that may auto-approve in autonomous mode.
var safeActionTypes = map[string]bool{
	"notification": true,
	"command":      true,
}

type PolicyEvaluator struct {
	assetSvc *asset.Service
}

func NewPolicyEvaluator(assetSvc *asset.Service) *PolicyEvaluator {
	return &PolicyEvaluator{assetSvc: assetSvc}
}

// Evaluate decides whether an action can auto-approve based on the
// asset's autonomy mode and the action type.
//
// manual     -> never auto-approve
// advisory   -> never auto-approve (AI suggests, operator approves)
// autonomous -> safe actions (notification, command) auto-approve;
//
//	risky actions (restart, config_change) require approval
func (e *PolicyEvaluator) Evaluate(ctx context.Context, a Action) (bool, string) {
	if a.AssetID == "" {
		return false, "no asset"
	}

	mode, err := e.assetSvc.GetAutonomyMode(ctx, a.AssetID)
	if err != nil {
		slog.Warn("policy: failed to read autonomy mode, defaulting to manual", "assetId", a.AssetID, "error", err)
		return false, "autonomy mode unavailable"
	}

	switch mode {
	case modeAutonomous:
		if safeActionTypes[a.Type] {
			return true, "auto-approved: safe action in autonomous mode"
		}
		return false, "requires approval: risky action in autonomous mode"
	case modeAdvisory:
		return false, "requires approval: advisory mode"
	case modeManual:
		return false, "requires approval: manual mode"
	default:
		return false, "unknown autonomy mode"
	}
}
