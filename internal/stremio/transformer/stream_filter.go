package stremio_transformer

import (
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
)

type Resolution string

func (r Resolution) Order() int64 {
	return getResolutionRank(string(r))
}

func (r Resolution) Normalize() string {
	return strings.ToLower(string(r))
}

type Quality string

func (q Quality) Order() int64 {
	return getQualityRank(string(q))
}

func (q Quality) Normalize() string {
	return strings.ToLower(string(q))
}

type Size string

func (s Size) Order() int64 {
	return getSizeRank(string(s))
}

func (s Size) Normalize() string {
	return strings.ToLower(string(s))
}

type ValuePatcher struct{}

func toOrderable(node ast.Node, converter string) ast.Node {
	return &ast.CallNode{
		Callee: &ast.MemberNode{
			Node: &ast.CallNode{
				Callee: &ast.IdentifierNode{
					Value: converter,
				},
				Arguments: []ast.Node{node},
			},
			Property: &ast.StringNode{Value: "Order"},
			Method:   true,
		},
		Arguments: []ast.Node{},
	}
}

var orderableConverter = map[string]string{
	"Resolution": "__Resolution__",
	"Quality":    "__Quality__",
	"Size":       "__Size__",
	"File.Size":  "__Size__",
}

func (ValuePatcher) Visit(node *ast.Node) {
	if bin, ok := (*node).(*ast.BinaryNode); ok {
		if _, ok := (bin.Left).(*ast.IdentifierNode); ok {
			if converter, exists := orderableConverter[bin.Left.String()]; exists {
				ast.Patch(&bin.Left, toOrderable(bin.Left, converter))
				ast.Patch(&bin.Right, toOrderable(bin.Right, converter))
			}
		} else if m, ok := (bin.Left).(*ast.MemberNode); ok && m.Node.String() == "File" && m.Property.String() == `"Size"` {
			if converter, exists := orderableConverter["File.Size"]; exists {
				ast.Patch(&bin.Left, toOrderable(bin.Left, converter))
				ast.Patch(&bin.Right, toOrderable(bin.Right, converter))
			}
		}

		if _, ok := (bin.Right).(*ast.IdentifierNode); ok {
			if converter, exists := orderableConverter[bin.Right.String()]; exists {
				ast.Patch(&bin.Left, toOrderable(bin.Left, converter))
				ast.Patch(&bin.Right, toOrderable(bin.Right, converter))
			}
		} else if m, ok := (bin.Right).(*ast.MemberNode); ok && m.Node.String() == "File" && m.Property.String() == `"Size"` {
			if converter, exists := orderableConverter["File.Size"]; exists {
				ast.Patch(&bin.Left, toOrderable(bin.Left, converter))
				ast.Patch(&bin.Right, toOrderable(bin.Right, converter))
			}
		}
	}
}

type StreamFilterEnv struct {
	*StreamExtractorResult
}

type StreamFilterBlob string

type StreamFilter struct {
	Blob    StreamFilterBlob
	program *vm.Program
}

func (sfb StreamFilterBlob) Parse() (*StreamFilter, error) {
	sf := &StreamFilter{
		Blob: sfb,
	}

	if sfb == "" {
		return sf, nil
	}

	program, err := expr.Compile(
		string(sfb),
		expr.Env(&StreamExtractorResult{}),
		expr.AsBool(),
		expr.AllowUndefinedVariables(),
		expr.Function("__Resolution__", func(val ...any) (any, error) {
			return Resolution(val[0].(string)), nil
		}, new(func(string) Resolution)),
		expr.Function("__Quality__", func(val ...any) (any, error) {
			return Quality(val[0].(string)), nil
		}, new(func(string) Quality)),
		expr.Function("__Size__", func(val ...any) (any, error) {
			return Size(val[0].(string)), nil
		}, new(func(string) Size)),
		expr.Patch(ValuePatcher{}),
	)
	if err != nil {
		return sf, err
	}

	sf.program = program
	return sf, nil
}

func (sf *StreamFilter) Match(r *StreamExtractorResult) bool {
	if sf == nil || sf.program == nil || r == nil {
		return true
	}

	output, err := expr.Run(sf.program, r)
	if err != nil {
		return true
	}

	return output.(bool)
}
