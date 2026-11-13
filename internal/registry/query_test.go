package registry

import "testing"

// A row names its source with the owner the index holds for it. The stored
// qualified name is the path, a colon and the name as the language qualifies
// it; the path is stripped as a prefix because a C++ name has colons too.
func TestQualifiedNameComesFromTheStoredQualifiedName(t *testing.T) {
	for _, tc := range []struct{ name, fqn, path, want string }{
		{"AddEdge", "internal/registry/writer.go:Writer.AddEdge", "internal/registry/writer.go", "Writer.AddEdge"},
		{"TestEdgeValidate", "pkg/model/model_test.go:TestEdgeValidate", "pkg/model/model_test.go", "TestEdgeValidate"},
		{"draw", "src/shape.cc:geo::Shape::draw", "src/shape.cc", "geo::Shape::draw"},
		{"f", "", "a.go", "f"},
		{"f", "elsewhere.go:T.f", "a.go", "f"},
		{"f", "a.go:", "a.go", "f"},
	} {
		if got := qualifiedName(tc.name, tc.fqn, tc.path); got != tc.want {
			t.Errorf("qualifiedName(%q, %q, %q) = %q, want %q", tc.name, tc.fqn, tc.path, got, tc.want)
		}
	}
}
