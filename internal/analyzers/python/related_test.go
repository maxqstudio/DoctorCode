package pythonanalysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestRelatedLocationsResolvePythonImportsAndExcludeShadows(t *testing.T) {
	root := t.TempDir()
	mustWritePython(t, root, "src/pkg/core.py", "def _target(flag):\n    if flag:\n        return True\n    else:\n        return False\n\ndef caller():\n    return _target(True)\n\ndef shadow():\n    _target = lambda value: value\n    return _target(False)\n")
	mustWritePython(t, root, "tests/test_core.py", "from pkg.core import _target as imported_target\n\ndef test_direct():\n    assert imported_target(False) is False\n")
	mustWritePython(t, root, "tests/test_module.py", "import pkg.core as core\n\ndef test_module():\n    assert core._target(True) is True\n")
	mustWritePython(t, root, "tests/test_shadow.py", "from pkg.core import _target as imported_target\n\ndef test_shadow():\n    imported_target = lambda value: value\n    assert imported_target(False) is False\n")
	mustWritePython(t, root, "other.py", "def _target(flag):\n    return flag\n\ndef caller():\n    return _target(True)\n")

	finding := model.Finding{Path: "src/pkg/core.py", LineStart: 2, LineEnd: 5}
	got, err := RelatedLocations(context.Background(), root, finding)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected source + direct-import test + module-attribute test references, got %#v", got)
	}
	if got[0].Path != "src/pkg/core.py" || got[0].Line != 8 || got[0].Kind != "reference" {
		t.Fatalf("unexpected source reference: %#v", got[0])
	}
	if got[1].Path != "tests/test_core.py" || got[1].Line != 4 || got[1].Kind != "test_reference" {
		t.Fatalf("unexpected direct-import test reference: %#v", got[1])
	}
	if got[2].Path != "tests/test_module.py" || got[2].Line != 4 || got[2].Kind != "test_reference" {
		t.Fatalf("unexpected module-attribute test reference: %#v", got[2])
	}
}

func TestRelatedLocationsReturnEmptyWhenFindingIsNotInsideModuleLevelFunction(t *testing.T) {
	root := t.TempDir()
	mustWritePython(t, root, "module.py", "API_TOKEN = \"not-a-real-secret-value\"\n")
	got, err := RelatedLocations(context.Background(), root, model.Finding{Path: "module.py", LineStart: 1, LineEnd: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no related function context, got %#v", got)
	}
}

func mustWritePython(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
