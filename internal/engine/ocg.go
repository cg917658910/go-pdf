package engine

import (
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type OCGSet struct {
	Fallback *types.IndirectRef
	Normal   *types.IndirectRef
}

func createOCG(ctx *model.Context, name string) *types.IndirectRef {
	d := types.Dict{
		"Type": types.Name("OCG"),
		"Name": types.StringLiteral(name),
	}
	ref, err := ctx.IndRefForNewObject(d)
	if err != nil {
		log.Info.Printf("Error creating OCG %s: %v\n", name, err)
	}
	return ref
}

func registerOCGs(ctx *model.Context) (*OCGSet, error) {
	return &OCGSet{
		Fallback: createOCG(ctx, "OCG_Fallback"),
		Normal:   createOCG(ctx, "OCG_Normal"),
	}, nil
}

func injectOCProperties(ctx *model.Context, ocgs *OCGSet) {
	ctx.RootDict["OCProperties"] = types.Dict{
		"OCGs": types.Array{
			*ocgs.Fallback,
			*ocgs.Normal,
		},
		"D": types.Dict{
			"Order": types.Array{
				*ocgs.Fallback,
				*ocgs.Normal,
			},
			"ON": types.Array{
				*ocgs.Fallback,
			},
			"OFF": types.Array{
				*ocgs.Normal,
			},
		},
	}
}

func injectOCGResources(ctx *model.Context, page types.Dict, ocgs *OCGSet) {
	res, _ := ctx.DereferenceDict(page["Resources"])
	if res == nil {
		res = types.Dict{}
		page["Resources"] = res
	}
	// Ensure Properties exists and map OCG names to indirect refs
	res["Properties"] = types.Dict{
		"OCG_Fallback": *ocgs.Fallback,
		"OCG_Normal":   *ocgs.Normal,
	}
	// Add Usage/Application entries to guide non-JS viewers (optional)
	// Not all viewers support Usage, but it can help some show/hide logic.
	if res["Properties"] == nil {
		res["Properties"] = types.Dict{}
	}
	// Add OCG to page resources' Optional Content if needed.
	page["Resources"] = res
}
