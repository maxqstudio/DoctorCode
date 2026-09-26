package detector

import (
	"context"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, root string) ([]model.Finding, error)
}
