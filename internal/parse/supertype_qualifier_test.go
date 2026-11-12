package parse

import (
	"context"
	"testing"
)

// TestSupertypeQualifier pins the receiver of a supertype reference: the
// qualifier as written, `.`-joined, and empty when the name is unqualified.
func TestSupertypeQualifier(t *testing.T) {
	cases := []struct {
		file, src string
		want      map[string]string // supertype name -> receiver
	}{
		{"a.scala", "class A extends java.io.OutputStream\nclass B extends C", map[string]string{"OutputStream": "java.io", "C": ""}},
		{"A.java", "class A extends java.io.OutputStream implements java.io.Closeable, Local {}", map[string]string{"OutputStream": "java.io", "Closeable": "java.io", "Local": ""}},
		{"a.kt", "class A : java.io.OutputStream(), Local", map[string]string{"OutputStream": "java.io", "Local": ""}},
		{"a.cs", "class A : System.IO.Stream, Local {}", map[string]string{"Stream": "System.IO", "Local": ""}},
		{"a.cpp", "class A : public std::ostream, public ns::Base<int>, public Local {};", map[string]string{"ostream": "std", "Base": "ns", "Local": ""}},
		{"a.php", "<?php\nclass A extends \\Ns\\Base {}\nclass B extends Local {}", map[string]string{"Base": "Ns", "Local": ""}},
		{"a.py", "class A(pkg.Base, pkg.sub.Gen[int], Local):\n    pass", map[string]string{"Base": "pkg", "Gen": "pkg.sub", "Local": ""}},
		{"a.rb", "class A < Mod::Base; end\nclass B < Local; end", map[string]string{"Base": "Mod", "Local": ""}},
		{"a.ts", "class A extends ns.Base implements Local {}", map[string]string{"Base": "ns", "Local": ""}},
		{"a.rs", "trait T: path::Trait + crate::other::Tr {}\ntrait U: super::V {}", map[string]string{"Trait": "path", "Tr": "other", "V": ""}},
	}
	reg := NewRegistry()
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			p := reg.ForFile(c.file)
			if p == nil {
				t.Fatalf("no parser for %s", c.file)
			}
			res, err := p.Parse(context.Background(), c.file, []byte(c.src))
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]string{}
			for _, r := range res.References {
				if isSupertypeRef(r.Kind) {
					got[r.Name] = r.Receiver
				}
			}
			for name, want := range c.want {
				if g, ok := got[name]; !ok || g != want {
					t.Errorf("%s: supertype %s receiver = %q (present %v), want %q; all: %v", c.file, name, g, ok, want, got)
				}
			}
		})
	}
}
