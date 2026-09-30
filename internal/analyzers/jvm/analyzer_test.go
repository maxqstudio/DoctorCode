package jvmanalysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/detector"
)

func TestDescriptorValid(t *testing.T) {
	if err := detector.ValidateDescriptor(New().Descriptor()); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzerFindsBoundedJavaAndKotlinRules(t *testing.T) {
	root := t.TempDir()
	javaSource := `class JavaFeature {
    private static final String API_TOKEN = "hardcoded-production-token-12345";
    boolean enabled(boolean flag) {
        if (flag) {
            return true;
        } else {
            return false;
        }
    }
    int classify(boolean flag) {
        if (flag) {
            return 1;
        } else if (flag) {
            return 2;
        }
        return 3;
    }
}
`
	kotlinSource := `class KotlinFeature {
    private val apiToken = "hardcoded-production-token-67890"
    fun enabled(flag: Boolean): Boolean {
        return if (flag) {
            true
        } else {
            false
        }
    }
    fun classify(flag: Boolean): Int {
        if (flag) {
            return 1
        } else if (flag) {
            return 2
        }
        return 3
    }
}
`
	if err := os.WriteFile(filepath.Join(root, "JavaFeature.java"), []byte(javaSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "KotlinFeature.kt"), []byte(kotlinSource), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, finding := range findings {
		counts[finding.RuleID]++
		if finding.SafeAutofix {
			t.Fatalf("JVM finding unexpectedly enables safe autofix: %#v", finding)
		}
		if strings.Contains(strings.Join(finding.Evidence, "\n"), "hardcoded-production-token") {
			t.Fatal("security evidence leaked credential literal")
		}
	}
	for _, rule := range []string{ruleSecurity, ruleSimplify, ruleLogic} {
		if counts[rule] != 2 {
			t.Fatalf("%s count=%d want=2 findings=%#v", rule, counts[rule], findings)
		}
	}
}

func TestMalformedJVMSyntaxFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
		src  string
	}{
		{name: "java", file: "Broken.java", src: "class Broken { boolean f( {\n"},
		{name: "kotlin", file: "Broken.kt", src: "class Broken { fun f( {\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, tc.file), []byte(tc.src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := New().Analyze(context.Background(), root)
			if err == nil || !strings.Contains(err.Error(), "syntax validation failed") {
				t.Fatalf("malformed %s must fail closed, got %v", tc.name, err)
			}
		})
	}
}

func TestBuildOutputsDoNotEstablishAvailability(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "build", "generated")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "Generated.java"), []byte("class Generated {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := New().Analyze(context.Background(), root)
	if !errors.Is(err, detector.ErrUnavailable) {
		t.Fatalf("build-only root error=%v want ErrUnavailable", err)
	}
}
