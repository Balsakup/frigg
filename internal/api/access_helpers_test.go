package api_test

import "github.com/danielgtaylor/huma/v2"

type hOp struct{ *huma.Operation }

func wrap(op *huma.Operation) *hOp {
	if op == nil {
		return nil
	}
	return &hOp{op}
}
