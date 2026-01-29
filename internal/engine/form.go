package engine

import (
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// injectTagField creates a probe field and per-page form widgets for Fallback/Normal/Expired.
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

	// create a single probe field used by JS to detect form support
	probe := types.Dict{
		"T":  types.StringLiteral("tag_probe"),
		"FT": types.Name("Tx"),
	}

	probeIR, err := ctx.IndRefForNewObject(probe)
	if err != nil {
		return err
	}

	if o, found := form.Find("Fields"); found {
		arr, err := ctx.DereferenceArray(o)
		if err == nil {
			arr = append(arr, *probeIR)
			form["Fields"] = arr
		} else {
			form["Fields"] = types.Array{*probeIR}
		}
	} else {
		form["Fields"] = types.Array{*probeIR}
	}

	// For every page create three invisible text fields (Fallback/Normal/Expired) with same position.
	for p := 1; p <= ctx.PageCount; p++ {
		pageDict, _, _, err := ctx.PageDict(p, false)
		if err != nil || pageDict == nil {
			continue
		}

		// choose a rect near top-left; caller should adjust layout as needed
		rect := types.Array{types.Integer(72), types.Integer(700), types.Integer(400), types.Integer(740)}

		names := []string{"_FG_Fallback", "_FG_Normal", "_FG_Expired"}

		// Create fields and widgets
		for _, nm := range names {
			f := types.Dict{
				"T":  types.StringLiteral(nm),
				"FT": types.Name("Tx"),
			}
			fir, err := ctx.IndRefForNewObject(f)
			if err != nil {
				continue
			}

			w := types.Dict{
				"Type":    types.Name("Annot"),
				"Subtype": types.Name("Widget"),
				"FT":      types.Name("Tx"),
				"T":       types.StringLiteral(nm),
				"Rect":    rect,
				"F":       types.Integer(4),
				"V":       types.StringLiteral(""),
			}
			wir, err := ctx.IndRefForNewObject(w)
			if err != nil {
				continue
			}

			// link widget via Kids
			if entry, found := ctx.FindTableEntryLight(int(fir.ObjectNumber)); found && entry != nil && entry.Object != nil {
				if fd, ok := entry.Object.(types.Dict); ok {
					fd["Kids"] = types.Array{*wir}
					entry.Object = fd
				}
			}

			// append to Fields
			if o, found := form.Find("Fields"); found {
				arr, err := ctx.DereferenceArray(o)
				if err == nil {
					arr = append(arr, *fir)
					form["Fields"] = arr
				} else {
					form["Fields"] = types.Array{*fir}
				}
			} else {
				form["Fields"] = types.Array{*fir}
			}

			// add widget to page Annots
			if ann, found := pageDict.Find("Annots"); found {
				arr, err := ctx.DereferenceArray(ann)
				if err == nil {
					arr = append(arr, *wir)
					pageDict["Annots"] = arr
				}
			} else {
				pageDict["Annots"] = types.Array{*wir}
			}
		}
	}

	ctx.RootDict["AcroForm"] = form
	return nil
}
