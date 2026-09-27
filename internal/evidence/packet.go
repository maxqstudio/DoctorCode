package evidence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

type Packet struct {
	SchemaVersion           int           `json:"schema_version"`
	Finding                 model.Finding `json:"finding"`
	SourceExcerpt           string        `json:"source_excerpt,omitempty"`
	SensitiveExcerptOmitted bool          `json:"sensitive_excerpt_omitted,omitempty"`
	BudgetBytes             int           `json:"budget_bytes"`
	Truncated               bool          `json:"truncated"`
}

func Build(root string, finding model.Finding, maxBytes int) (Packet, error) {
	if maxBytes < 512 {
		return Packet{}, errors.New("max-bytes must be at least 512")
	}

	packet := Packet{
		SchemaVersion: 1,
		Finding:       finding,
		BudgetBytes:   maxBytes,
	}

	if finding.Category == model.CategorySecurity {
		packet.SensitiveExcerptOmitted = true
		return fit(packet, maxBytes)
	}

	for _, radius := range []int{3, 1, 0} {
		excerpt, err := sourceExcerpt(root, finding, radius)
		if err != nil {
			return Packet{}, err
		}
		packet.SourceExcerpt = excerpt
		packet.Truncated = radius < 3
		if encodedSize(packet) <= maxBytes {
			return packet, nil
		}
	}

	packet.SourceExcerpt = ""
	packet.Truncated = true
	return fit(packet, maxBytes)
}

func fit(packet Packet, maxBytes int) (Packet, error) {
	if encodedSize(packet) > maxBytes {
		return Packet{}, fmt.Errorf("finding metadata exceeds max-bytes=%d", maxBytes)
	}
	return packet, nil
}

func encodedSize(packet Packet) int {
	data, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return 1 << 30
	}
	return len(data) + 1
}

func sourceExcerpt(root string, finding model.Finding, radius int) (string, error) {
	path, err := resolveFindingPath(root, finding.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return "", nil
	}

	start := finding.LineStart
	if start < 1 {
		start = 1
	}
	end := finding.LineEnd
	if end < start {
		end = start
	}
	start -= radius
	if start < 1 {
		start = 1
	}
	end += radius
	if end > len(lines) {
		end = len(lines)
	}

	var out strings.Builder
	for line := start; line <= end; line++ {
		fmt.Fprintf(&out, "%d: %s\n", line, lines[line-1])
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}

func resolveFindingPath(root, findingPath string) (string, error) {
	if strings.TrimSpace(findingPath) == "" {
		return "", errors.New("finding path is empty")
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}

	localPath := filepath.FromSlash(findingPath)
	if filepath.IsAbs(localPath) {
		return "", fmt.Errorf("finding path escapes repository root: %s", findingPath)
	}
	candidateAbs, err := filepath.Abs(filepath.Join(rootResolved, localPath))
	if err != nil {
		return "", fmt.Errorf("resolve finding path: %w", err)
	}
	candidateResolved, err := filepath.EvalSymlinks(candidateAbs)
	if err != nil {
		return "", fmt.Errorf("resolve finding path: %w", err)
	}

	rel, err := filepath.Rel(rootResolved, candidateResolved)
	if err != nil {
		return "", fmt.Errorf("compare finding path with repository root: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("finding path escapes repository root: %s", findingPath)
	}
	return candidateResolved, nil
}
