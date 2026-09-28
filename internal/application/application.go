package application

import (
	"context"
	"fmt"

	"github.com/maxqstudio/DoctorCode/internal/engine"
	"github.com/maxqstudio/DoctorCode/internal/evidence"
	"github.com/maxqstudio/DoctorCode/internal/model"
	"github.com/maxqstudio/DoctorCode/internal/verification"
)

func Audit(ctx context.Context, root string) (model.AuditResult, error) {
	return engine.Default().Audit(ctx, root)
}

func Context(ctx context.Context, root, findingID string, maxBytes int) (evidence.Packet, error) {
	audit, err := Audit(ctx, root)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("audit failed: %w", err)
	}
	finding, ok := findFindingByID(audit.Findings, findingID)
	if !ok {
		return evidence.Packet{}, fmt.Errorf("finding %q not found", findingID)
	}
	packet, err := evidence.Build(audit.Root, finding, maxBytes)
	if err != nil {
		return evidence.Packet{}, fmt.Errorf("build evidence packet: %w", err)
	}
	return packet, nil
}

func Contract(ctx context.Context, root, findingID string) (verification.Contract, error) {
	audit, err := Audit(ctx, root)
	if err != nil {
		return verification.Contract{}, fmt.Errorf("audit failed: %w", err)
	}
	contract, err := verification.BuildContract(audit, findingID)
	if err != nil {
		return verification.Contract{}, fmt.Errorf("build verification contract: %w", err)
	}
	return contract, nil
}

func Verify(ctx context.Context, root string, contract verification.Contract) (verification.Result, error) {
	audit, err := Audit(ctx, root)
	if err != nil {
		return verification.Result{}, fmt.Errorf("audit failed: %w", err)
	}
	result, err := verification.Verify(contract, audit)
	if err != nil {
		return verification.Result{}, fmt.Errorf("verify contract: %w", err)
	}
	return result, nil
}

func findFindingByID(findings []model.Finding, findingID string) (model.Finding, bool) {
	for _, finding := range findings {
		if finding.ID == findingID {
			return finding, true
		}
	}
	return model.Finding{}, false
}
