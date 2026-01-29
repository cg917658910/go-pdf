package engine

import (
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type OCGSet struct {
	Fallback *types.IndirectRef
	Normal   *types.IndirectRef
	Expired  *types.IndirectRef
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
		Expired:  createOCG(ctx, "OCG_Expired"),
	}, nil
}

func injectOCProperties(ctx *model.Context, ocgs *OCGSet) {
	ctx.RootDict["OCProperties"] = types.Dict{
		"OCGs": types.Array{
			*ocgs.Fallback,
			*ocgs.Normal,
			*ocgs.Expired,
		},
		"D": types.Dict{
			"Order": types.Array{
				*ocgs.Fallback,
				*ocgs.Normal,
				*ocgs.Expired,
			},
			"ON": types.Array{
				*ocgs.Fallback,
			},
			"OFF": types.Array{
				*ocgs.Normal,
				*ocgs.Expired,
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
	res["Properties"] = types.Dict{
		"OCG_Fallback": *ocgs.Fallback,
		"OCG_Normal":   *ocgs.Normal,
		"OCG_Expired":  *ocgs.Expired,
	}
}
