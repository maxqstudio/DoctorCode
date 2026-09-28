package doctorcodemcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/application"
	"github.com/maxqstudio/DoctorCode/internal/evidence"
	"github.com/maxqstudio/DoctorCode/internal/model"
	"github.com/maxqstudio/DoctorCode/internal/verification"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultMaxFindings = 100
	maxMaxFindings     = 1000
	defaultMaxBytes              = 4096
	maxMaxBytes                  = 64 * 1024
	maxVerificationContractBytes = 1 << 20
)

type auditInput struct {
	MaxFindings int `json:"max_findings,omitempty" jsonschema:"maximum findings to return; zero uses the bounded default"`
}

type auditOutput struct {
	Analyzers        []string        `json:"analyzers"`
	TotalFindings    int             `json:"total_findings"`
	ReturnedFindings int             `json:"returned_findings"`
	Findings         []model.Finding `json:"findings"`
}

type contextInput struct {
	FindingID string `json:"finding_id" jsonschema:"exact current DoctorCode finding identifier"`
	MaxBytes  int    `json:"max_bytes,omitempty" jsonschema:"maximum encoded evidence packet bytes; zero uses the bounded default"`
}

type contractInput struct {
	FindingID string `json:"finding_id" jsonschema:"exact current DoctorCode finding identifier"`
}

type verifyInput struct {
	Contract map[string]any `json:"contract" jsonschema:"verification contract returned by doctorcode_contract"`
}

func NewServer(root string) (*mcp.Server, error) {
	boundRoot, err := bindRoot(root)
	if err != nil {
		return nil, err
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "doctorcode-mcp", Version: "0.1.0-dev"},
		nil,
	)

	addAuditTool(server, boundRoot)
	addContextTool(server, boundRoot)
	addContractTool(server, boundRoot)
	addVerifyTool(server, boundRoot)
	return server, nil
}

func bindRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("startup repository root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve startup repository root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve startup repository root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat startup repository root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("startup repository root is not a directory: %s", resolved)
	}
	return resolved, nil
}

func addAuditTool(server *mcp.Server, root string) {
	mcp.AddTool(server, readOnlyTool(
		"doctorcode_audit",
		"Audit the bound repository with DoctorCode deterministic analyzers.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, in auditInput) (*mcp.CallToolResult, auditOutput, error) {
		maxFindings, err := boundedMaxFindings(in.MaxFindings)
		if err != nil {
			return nil, auditOutput{}, err
		}
		audit, err := application.Audit(ctx, root)
		if err != nil {
			return nil, auditOutput{}, err
		}
		total := len(audit.Findings)
		findings := audit.Findings
		if len(findings) > maxFindings {
			findings = findings[:maxFindings]
		}
		return nil, auditOutput{
			Analyzers:        audit.Analyzers,
			TotalFindings:    total,
			ReturnedFindings: len(findings),
			Findings:         findings,
		}, nil
	})
}

func addContextTool(server *mcp.Server, root string) {
	mcp.AddTool(server, readOnlyTool(
		"doctorcode_context",
		"Build a bounded evidence packet for one exact current DoctorCode finding in the bound repository.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, in contextInput) (*mcp.CallToolResult, evidence.Packet, error) {
		if strings.TrimSpace(in.FindingID) == "" {
			return nil, evidence.Packet{}, errors.New("finding_id is required")
		}
		maxBytes, err := boundedMaxBytes(in.MaxBytes)
		if err != nil {
			return nil, evidence.Packet{}, err
		}
		packet, err := application.Context(ctx, root, in.FindingID, maxBytes)
		if err != nil {
			return nil, evidence.Packet{}, err
		}
		return nil, packet, nil
	})
}

func addContractTool(server *mcp.Server, root string) {
	mcp.AddTool(server, readOnlyTool(
		"doctorcode_contract",
		"Freeze the deterministic pre-repair verification contract for one exact current finding.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, in contractInput) (*mcp.CallToolResult, verification.Contract, error) {
		if strings.TrimSpace(in.FindingID) == "" {
			return nil, verification.Contract{}, errors.New("finding_id is required")
		}
		contract, err := application.Contract(ctx, root, in.FindingID)
		if err != nil {
			return nil, verification.Contract{}, err
		}
		return nil, contract, nil
	})
}

func addVerifyTool(server *mcp.Server, root string) {
	mcp.AddTool(server, readOnlyTool(
		"doctorcode_verify",
		"Re-audit the bound repository against a pre-repair DoctorCode verification contract.",
	), func(ctx context.Context, _ *mcp.CallToolRequest, in verifyInput) (*mcp.CallToolResult, verification.Result, error) {
		contract, err := decodeContract(in.Contract)
		if err != nil {
			return nil, verification.Result{}, err
		}
		result, err := application.Verify(ctx, root, contract)
		if err != nil {
			return nil, verification.Result{}, err
		}
		if !result.Passed {
			return &mcp.CallToolResult{IsError: true}, result, nil
		}
		return nil, result, nil
	})
}

func readOnlyTool(name, description string) *mcp.Tool {
	openWorld := false
	destructive := false
	return &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			OpenWorldHint:   &openWorld,
			DestructiveHint: &destructive,
		},
	}
}

func boundedMaxFindings(value int) (int, error) {
	if value == 0 {
		return defaultMaxFindings, nil
	}
	if value < 1 || value > maxMaxFindings {
		return 0, fmt.Errorf("max_findings must be between 1 and %d", maxMaxFindings)
	}
	return value, nil
}

func boundedMaxBytes(value int) (int, error) {
	if value == 0 {
		return defaultMaxBytes, nil
	}
	if value < 512 || value > maxMaxBytes {
		return 0, fmt.Errorf("max_bytes must be between 512 and %d", maxMaxBytes)
	}
	return value, nil
}

func decodeContract(value map[string]any) (verification.Contract, error) {
	if value == nil {
		return verification.Contract{}, errors.New("contract is required")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return verification.Contract{}, fmt.Errorf("encode verification contract: %w", err)
	}
	if len(data) > maxVerificationContractBytes {
		return verification.Contract{}, fmt.Errorf(
			"verification contract exceeds %d bytes",
			maxVerificationContractBytes,
		)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var contract verification.Contract
	if err := decoder.Decode(&contract); err != nil {
		return verification.Contract{}, fmt.Errorf("decode verification contract: %w", err)
	}
	return contract, nil
}
