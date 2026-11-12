package parse

import (
	"github.com/satyendrakumarsingh/repocortex/pkg/model"

	ts "github.com/odvcencio/gotreesitter"
)

// # Import bindings — which local names an import statement creates
//
// The import query records that a file imports `../core/util.js`. It does not
// record that the file then calls that module `util`, and a resolver that is
// not told has to guess what `util.joinValues(...)` is qualified by. The guess
// it made was a comparison of the receiver's name with a file's base name, and
// on zod — which has a `util.ts` in both its v3 and its v4 tree — that bound
// 121 v4 calls to the v3 file.
//
// An ECMAScript import statement states the binding outright, in three forms,
// and each is recorded the way pyscope.go records the Python equivalents:
//
//	import * as util from "./util.js"     Path "./util.js", Alias "util"
//	import def from "./helpers"           Path "./helpers", Alias ".def"
//	import { a, b as c } from "./m"       Path "./m", Alias ".a" and ".c"
//
// A namespace import names the MODULE, so it is a module alias, exactly like
// Python's `import x as y`. A default or a named import names something INSIDE
// the module, so it is a member, exactly like `from x import y` — the alias
// carries memberAliasPrefix. A renamed member is recorded under its LOCAL name:
// what the resolver needs from it is that a receiver spelled `c` came from
// that module, and where the module is not in the repository that is the whole
// answer.

// importBindingLanguages are the languages whose import statements this file
// reads. They share one grammar family and one statement shape.
var importBindingLanguages = map[string]bool{
	"javascript": true, "typescript": true, "tsx": true,
}

// importedBindings returns the local names a file's import statements bind,
// as Import rows to merge with the import query's. Nil for a language it does
// not read and for a tree it cannot walk.
func (p *genericParser) importedBindings(tree *ts.Tree, src []byte) (out []Import) {
	if tree == nil || !importBindingLanguages[p.entry.Name] {
		return nil
	}
	defer func() { _ = recover() }()

	root := tree.RootNode()
	if root == nil {
		return nil
	}
	lang := p.lang
	add := func(path, alias string, at *ts.Node) {
		if path == "" || alias == "" || alias == memberAliasPrefix || at == nil {
			return
		}
		imp := Import{Path: path, Alias: alias}
		imp.Range = model.Range{
			Start: model.Position{Line: int(at.StartPoint().Row) + 1, Col: int(at.StartPoint().Column)},
			End:   model.Position{Line: int(at.EndPoint().Row) + 1, Col: int(at.EndPoint().Column)},
		}
		out = append(out, imp)
	}

	for i := 0; i < root.ChildCount(); i++ {
		stmt := root.Child(i)
		if stmt == nil || !stmt.IsNamed() || stmt.Type(lang) != "import_statement" {
			continue
		}
		source := stmt.ChildByFieldName("source", lang)
		if source == nil {
			continue
		}
		path := unquote(source.Text(src))
		for j := 0; j < stmt.ChildCount(); j++ {
			clause := stmt.Child(j)
			if clause == nil || !clause.IsNamed() || clause.Type(lang) != "import_clause" {
				continue
			}
			for k := 0; k < clause.ChildCount(); k++ {
				c := clause.Child(k)
				if c == nil || !c.IsNamed() {
					continue
				}
				switch c.Type(lang) {
				case "identifier":
					// The default import.
					add(path, memberAliasPrefix+c.Text(src), c)
				case "namespace_import":
					for m := 0; m < c.ChildCount(); m++ {
						if id := c.Child(m); id != nil && id.IsNamed() && id.Type(lang) == "identifier" {
							add(path, id.Text(src), id)
						}
					}
				case "named_imports":
					for m := 0; m < c.ChildCount(); m++ {
						spec := c.Child(m)
						if spec == nil || !spec.IsNamed() || spec.Type(lang) != "import_specifier" {
							continue
						}
						local := spec.ChildByFieldName("alias", lang)
						if local == nil {
							local = spec.ChildByFieldName("name", lang)
						}
						if local != nil {
							add(path, memberAliasPrefix+local.Text(src), local)
						}
					}
				}
			}
		}
	}
	return out
}
