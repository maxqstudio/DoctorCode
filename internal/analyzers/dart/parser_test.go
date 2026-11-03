package dartanalysis

import "testing"

func TestParseSyntaxAcceptsValidDart(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "basic class",
			source: `class Sample {
  bool isReady(bool value) {
    return value;
  }
}`,
		},
		{
			name: "async nullable",
			source: `Future<String?> loadAsync(String? value) async {
  await Future<void>.delayed(Duration.zero);
  return value;
}`,
		},
		{
			name: "flutter shaped syntax",
			source: `class FeatureWidget extends StatelessWidget {
  const FeatureWidget({super.key});

  @override
  Widget build(BuildContext context) {
    return const SizedBox.shrink();
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := parseSyntax([]byte(tt.source))
			if err != nil {
				t.Fatalf("parseSyntax() error = %v", err)
			}
			if doc == nil {
				t.Fatal("parseSyntax() returned nil document")
			}
			if doc.language == nil {
				t.Fatal("parseSyntax() returned nil language")
			}
			if doc.tree == nil {
				t.Fatal("parseSyntax() returned nil tree")
			}
			root := doc.tree.RootNode()
			if root == nil {
				t.Fatal("parseSyntax() returned tree with nil root")
			}
			if root.HasErrorOrMissing() {
				t.Fatal("parseSyntax() accepted tree containing ERROR or MISSING nodes")
			}
		})
	}
}

func TestParseSyntaxRejectsMalformedDart(t *testing.T) {
	source := []byte(`class Broken {
  bool run( {
    return true;
  }
}`)

	if doc, err := parseSyntax(source); err == nil {
		t.Fatalf("parseSyntax() accepted malformed Dart with document %#v", doc)
	}
}
