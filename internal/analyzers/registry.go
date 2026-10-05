package analyzers

import (
	"fmt"

	cppanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/cpp"
	goanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/golang"
	javascriptanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/javascript"
	jvmanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/jvm"
	pythonanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/python"
	rustanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/rust"
	"github.com/maxqstudio/DoctorCode/internal/detector"
)

func Default() []detector.Analyzer {
	return []detector.Analyzer{
		goanalysis.New(),
		pythonanalysis.New(),
		javascriptanalysis.New(),
		rustanalysis.New(),
		jvmanalysis.New(),
		cppanalysis.New(),
	}
}

func Descriptors() []detector.Descriptor {
	analyzers := Default()
	out := make([]detector.Descriptor, 0, len(analyzers))
	for _, analyzer := range analyzers {
		out = append(out, analyzer.Descriptor())
	}
	return out
}

func ValidateDefault() error {
	seenAnalyzers := map[string]bool{}
	seenRules := map[string]string{}
	for _, analyzer := range Default() {
		desc := analyzer.Descriptor()
		if desc.ID != analyzer.Name() {
			return fmt.Errorf("descriptor id %q does not match analyzer name %q", desc.ID, analyzer.Name())
		}
		if err := detector.ValidateDescriptor(desc); err != nil {
			return fmt.Errorf("%s descriptor: %w", analyzer.Name(), err)
		}
		if seenAnalyzers[desc.ID] {
			return fmt.Errorf("duplicate analyzer id %q", desc.ID)
		}
		seenAnalyzers[desc.ID] = true
		for _, rule := range desc.Rules {
			if owner, exists := seenRules[rule.ID]; exists {
				return fmt.Errorf("duplicate rule id %q declared by %s and %s", rule.ID, owner, desc.ID)
			}
			seenRules[rule.ID] = desc.ID
		}
	}
	return nil
}
