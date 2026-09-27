package detector

import (
	"context"
	"errors"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

var ErrUnavailable = errors.New("analyzer unavailable")

type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, root string) ([]model.Finding, error)
}
