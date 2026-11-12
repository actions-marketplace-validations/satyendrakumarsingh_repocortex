package parse

import (
	"regexp"
	"strings"
)

// HCL (Terraform, OpenTofu, Packer, Terragrunt, Nomad). What the index holds,
// see fixtures/langs/hcl/expected.tsv.
//
// The inferred query tagged every block at every depth and named it by its
// last label, or by its keyword when it has none. In a Terraform file that
// made `ingress`, `tags`, `provisioner "file"` and `locals` definitions (60% of
// the symbols were not), and gave a resource and a variable of the same name
// one name. It is replaced.
//
// A definition is a labelled block at the top level of the file, or an entry of
// a top-level `locals` block. A block nested in another block, and a block with
// no label (`terraform`, `locals`, `moved`, `import`), is configuration of
// something else.
//
// A block is named the way Terraform refers to it, so that two things of one
// name in different roles cannot collide:
//
//	resource "aws_instance" "web"   aws_instance.web      class
//	data "aws_ami" "ubuntu"         data.aws_ami.ubuntu    class
//	module "network"                module.network         package
//	provider "aws"                  provider.aws           package
//	provider "aws" { alias = "w" }  provider.aws.w         package
//	variable "region"               var.region             variable
//	output "id"                     output.id              constant
//	locals { name = ... }           local.name             constant
//	check "health"                  check.health           class
//
// Any other labelled top-level block (Packer `source`, Terragrunt
// `dependency`, Nomad `job`) is its keyword and labels joined by dots, a class.
//
// A query can capture one node as the name and the two labels are two nodes, so
// the block itself is captured as both @name and the definition; the name
// correction registered below receives the text of the block, reads its header
// (or the `name =` of a locals entry) and builds the name.
func init() {
	registerDefinitions("hcl", hclDefinitions, true)
	registerNameFix("hcl", hclName)
}

// hclLabelled is a top-level block with at least one label. HEAD stands for
// the predicate that selects the block by its keyword.
const hclLabelled = `(config_file (body (block . (identifier) @_kw [(string_lit) (identifier)] (block_start) HEAD) @name @definition.KIND))`

func hclPattern(kind, head string) string {
	return strings.NewReplacer("KIND", kind, "HEAD", head).Replace(hclLabelled)
}

var hclDefinitions = strings.Join([]string{
	hclPattern("class", `(#any-of? @_kw "resource" "data")`),
	hclPattern("package", `(#any-of? @_kw "module" "provider")`),
	hclPattern("variable", `(#eq? @_kw "variable")`),
	hclPattern("constant", `(#eq? @_kw "output")`),
	hclPattern("class", `(#not-any-of? @_kw "resource" "data" "module" "provider" "variable" "output" "locals")`),
	// An entry of a top-level locals block.
	`(config_file (body (block . (identifier) @_kw . (block_start) (body (attribute) @name @definition.constant)) (#eq? @_kw "locals")))`,
}, "\n")

var (
	// hclHeader reads a block's keyword and labels. A label is a quoted string
	// or a bare identifier.
	hclHeader = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)((?:\s+(?:"[^"]*"|[A-Za-z_][A-Za-z0-9_-]*))*)\s*\{`)
	hclLabel  = regexp.MustCompile(`"([^"]*)"|[A-Za-z_][A-Za-z0-9_-]*`)
	hclEntry  = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\s*=`)
	hclAlias  = regexp.MustCompile(`(?m)^\s*alias\s*=\s*"([^"]*)"`)
)

// hclName turns the text of a captured definition into its name.
func hclName(text string) string {
	if m := hclEntry.FindStringSubmatch(text); m != nil {
		return "local." + m[1]
	}
	m := hclHeader.FindStringSubmatch(text)
	if m == nil {
		// A header the pattern cannot read (a comment between the labels):
		// better its first line than the text of the whole block.
		first, _, _ := strings.Cut(text, "\n")
		return strings.TrimSpace(first)
	}
	keyword := m[1]
	var labels []string
	for _, l := range hclLabel.FindAllStringSubmatch(m[2], -1) {
		if l[1] != "" || strings.HasPrefix(l[0], `"`) {
			labels = append(labels, l[1])
		} else {
			labels = append(labels, l[0])
		}
	}
	switch keyword {
	case "resource":
		// The resource type is the first label, and is not a keyword.
	case "variable":
		labels = append([]string{"var"}, labels...)
	case "provider":
		labels = append([]string{"provider"}, labels...)
		if a := hclAlias.FindStringSubmatch(text); a != nil {
			labels = append(labels, a[1])
		}
	default:
		// data, module, output and every other block keep their keyword.
		labels = append([]string{keyword}, labels...)
	}
	return strings.Join(labels, ".")
}
