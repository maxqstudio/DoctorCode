package realworld

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dartanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/dart"
)

const m26FlutterSourceSHA = "21c71902bb2c7da3cce328ba5946cba2f35a660a"

func TestM26PinnedFlutterExamples(t *testing.T) {
	src := strings.TrimSpace(os.Getenv("DOCTORCODE_M26_FLUTTER_SAMPLES"))
	if src == "" {
		if os.Getenv("DOCTORCODE_M26_REQUIRE") == "1" {
			t.Fatal("required pinned Flutter checkout not provided")
		}
		t.Skip("M26 public Dart evidence is validated by its dedicated GitHub Actions workflow")
	}
	root, err := resolveSourceRoot(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyCheckoutSHA(root, m26FlutterSourceSHA); err != nil {
		t.Fatalf("pinned flutter/samples provenance failed: %v", err)
	}
	for _, tc := range []struct {
		name              string
		dartFiles         int
		knownParseFailure string
	}{
		{"form_app", 8, ""},
		{"navigation_and_routing", 19, "lib/src/screens/settings.dart"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			appRoot := filepath.Join(root, tc.name)
			if _, err := os.Stat(filepath.Join(appRoot, "pubspec.yaml")); err != nil {
				t.Fatalf("public sample pubspec missing: %v", err)
			}
			count := 0
			err := filepath.WalkDir(appRoot, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".dart") {
					count++
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if count != tc.dartFiles {
				t.Fatalf("pinned public sample Dart files=%d want=%d", count, tc.dartFiles)
			}
			findings, err := dartanalysis.New().Analyze(context.Background(), appRoot)
			if tc.knownParseFailure == "" {
				if err != nil {
					t.Fatalf("strict real-world Dart AST parse failed: %v", err)
				}
				if len(findings) != 0 {
					t.Fatalf("unexpected unreviewed findings in bounded public sample: %#v", findings)
				}
				t.Logf("PUBLIC_DART_SOURCE_PASS: flutter/samples@%s/%s (%d files, 0 observed findings)", m26FlutterSourceSHA, tc.name, count)
				return
			}
			// Unknown grammar semantics must remain fail-closed, not silently skipped.
			if err == nil {
				t.Fatalf("known unsupported public Dart syntax unexpectedly passed without updated evidence: %s", tc.knownParseFailure)
			}
			if !strings.Contains(filepath.ToSlash(err.Error()), tc.knownParseFailure) || !strings.Contains(err.Error(), "ERROR or MISSING") {
				t.Fatalf("unexpected error instead of bounded fail-closed evidence: %v", err)
			}
			if len(findings) != 0 {
				t.Fatalf("fail-closed analysis must not return partial findings: %#v", findings)
			}
			t.Logf("KNOWN_NOT_PROVEN: flutter/samples@%s/%s %s", m26FlutterSourceSHA, tc.name, tc.knownParseFailure)
		})
	}
}
