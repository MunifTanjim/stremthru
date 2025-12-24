package stremio_transformer

import (
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

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
