package csharpanalysis

import "testing"

func TestParseSyntaxAcceptsValidCSharp(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "basic class",
			source: `namespace DoctorCode {
	internal sealed class Sample {
		public bool IsReady(bool value) {
			return value;
		}
	}
}`,
		},
		{
			name: "async nullable",
			source: `#nullable enable
using System.Threading.Tasks;

namespace DoctorCode {
	internal sealed class Sample {
		public async Task<string?> LoadAsync(string? value) {
			await Task.Yield();
			return value;
		}
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

func TestParseSyntaxRejectsMalformedCSharp(t *testing.T) {
	source := []byte(`namespace DoctorCode {
	internal sealed class Broken {
		public void Run( {
	}
}`)

	if doc, err := parseSyntax(source); err == nil {
		t.Fatalf("parseSyntax() accepted malformed C# with document %#v", doc)
	}
}
