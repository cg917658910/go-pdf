package engine

import (
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func injectTagField(ctx *model.Context) error {
	// Ensure AcroForm dict exists.
	var form types.Dict
	if o, found := ctx.RootDict.Find("AcroForm"); found {
		if d, ok := o.(types.Dict); ok {
			form = d
		}
	}
	if form == nil {
		form = types.Dict{}
		ctx.RootDict["AcroForm"] = form
	}

	// create field dict
	f := types.Dict{
		"T":  types.StringLiteral("tag_probe"),
		"FT": types.Name("Tx"),
	}

	ir, err := ctx.IndRefForNewObject(f)
	if err != nil {
		return err
	}

	// append to Fields array
	if o, found := form.Find("Fields"); found {
		arr, err := ctx.DereferenceArray(o)
		if err == nil {
			arr = append(arr, *ir)
			form["Fields"] = arr
		} else {
			form["Fields"] = types.Array{*ir}
		}
	} else {
		form["Fields"] = types.Array{*ir}
	}

	ctx.RootDict["AcroForm"] = form
	return nil
}
