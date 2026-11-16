package infer

import "testing"

func TestLayoutProjectOf(t *testing.T) {
	files := []string{
		"polyglot/backend-go/go.mod",
		"polyglot/backend-go/cmd/main.go",
		"polyglot/worker-python/pyproject.toml",
		"polyglot/worker-python/tasks.py",
		"polyglot/web-ts/package.json",
		"polyglot/web-ts/src/index.ts",
		"polyglot/mobile-dart/pubspec.yaml",
		"polyglot/mobile-dart/lib/main.dart",
		"polyglot/billing-php/composer.json",
		"polyglot/billing-php/src/Charge.php",
		"polyglot/core-rust/Cargo.toml",
		"polyglot/core-rust/src/lib.rs",
		"polyglot/dotnet-app/App.csproj",
		"polyglot/dotnet-app/Program.cs",
		"root-file.go",
	}

	layout := NewLayout(files, nil)

	cases := []struct {
		path string
		want string
	}{
		{"polyglot/backend-go/cmd/main.go", "polyglot/backend-go"},
		{"polyglot/worker-python/tasks.py", "polyglot/worker-python"},
		{"polyglot/web-ts/src/index.ts", "polyglot/web-ts"},
		{"polyglot/mobile-dart/lib/main.dart", "polyglot/mobile-dart"},
		{"polyglot/billing-php/src/Charge.php", "polyglot/billing-php"},
		{"polyglot/core-rust/src/lib.rs", "polyglot/core-rust"},
		{"polyglot/dotnet-app/Program.cs", "polyglot/dotnet-app"},
		{"root-file.go", ""},
		{"unknown/dir/file.py", ""},
	}

	for _, tc := range cases {
		got := layout.ProjectOf(tc.path)
		if got != tc.want {
			t.Errorf("ProjectOf(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
