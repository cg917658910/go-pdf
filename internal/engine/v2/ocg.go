package engine

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func ensureOCGs(ctx *model.Context) (types.IndirectRef, types.IndirectRef) {
	normal := newOCG(ctx, "OCG_Normal")
	fallback := newOCG(ctx, "OCG_Fallback")

	ctx.RootDict["OCProperties"] = types.Dict{
		"OCGs": types.Array{normal, fallback},
		"D": types.Dict{
			"ON":  types.Array{fallback}, // 默认只显示 Fallback
			"OFF": types.Array{normal},
		},
	}

	return normal, fallback
}

func ensureOCGNormal(ctx *model.Context) types.IndirectRef {
	ocg := types.Dict{
		"Type": types.Name("OCG"),
		"Name": types.StringLiteral("OCG_Normal"),
	}

	ocgRef, err := ctx.IndRefForNewObject(ocg)
	if err != nil {
		fmt.Printf("Error creating OCG_Normal: %v\n", err)
	}

	ctx.RootDict["OCProperties"] = types.Dict{
		"OCGs": types.Array{*ocgRef},
		"D": types.Dict{
			"OFF": types.Array{*ocgRef}, // ⭐ 默认关闭
		},
	}

	return *ocgRef
}

func newOCG(ctx *model.Context, name string) types.IndirectRef {
	d := types.Dict{
		"Type": types.Name("OCG"),
		"Name": types.StringLiteral(name),
	}
	ref, err := ctx.IndRefForNewObject(d)
	if err != nil {
		fmt.Printf("Error creating OCG %s: %v\n", name, err)
	}
	return *ref
}
