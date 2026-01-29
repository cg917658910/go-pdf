package newengine

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// createFullPageFallbackWidget creates a full-page Widget annotation covering the page MediaBox with an AP that draws an opaque rectangle and text.
func createFullPageFallbackWidget(ctx *model.Context, pageNr int, fallbackText string) error {
	pageDict, _, _, err := ctx.PageDict(pageNr, false)
	if err != nil || pageDict == nil {
		return err
	}

	// determine MediaBox for rect
	rect := types.Array{types.Integer(0), types.Integer(0), types.Integer(612), types.Integer(792)}
	if mb, found := pageDict.Find("MediaBox"); found {
		if arr, err := ctx.DereferenceArray(mb); err == nil && len(arr) == 4 {
			rect = types.Array{arr[0], arr[1], arr[2], arr[3]}
		}
	}

	name := fmt.Sprintf("fallback_p%d", pageNr)

	// field dict
	f := types.Dict{
		"T":  types.StringLiteral(name),
		"FT": types.Name("Tx"),
	}
	fir, err := ctx.IndRefForNewObject(f)
	if err != nil {
		return err
	}

	// appearance stream: opaque white rectangle then centered text
	w := 612.0
	h := 792.0
	if len(rect) == 4 {
		if rx, ok := rect[2].(types.Integer); ok {
			w = float64(rx)
		}
		if ry, ok := rect[3].(types.Integer); ok {
			h = float64(ry)
		}
	}
	// Draw white rectangle, center text roughly vertically and horizontally
	ap := fmt.Sprintf("q 1 1 1 rg 0 0 %f %f re f Q BT /F1 36 Tf %f %f Td (%s) Tj ET", w, h, w/2-100, h/2, escape(fallbackText))

	sd, err := ctx.NewStreamDictForBuf([]byte(ap))
	if err != nil {
		return err
	}
	if err := sd.Encode(); err != nil {
		return err
	}
	apIr, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	// widget
	wDict := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"Rect":    rect,
		"F":       types.Integer(4),
		"T":       types.StringLiteral(name),
		"FT":      types.Name("Tx"),
		"AP":      types.Dict{"N": *apIr},
	}
	wir, err := ctx.IndRefForNewObject(wDict)
	if err != nil {
		return err
	}

	// link widget via Kids
	if entry, found := ctx.FindTableEntryLight(int(fir.ObjectNumber)); found && entry != nil && entry.Object != nil {
		if fd, ok := entry.Object.(types.Dict); ok {
			fd["Kids"] = types.Array{*wir}
			entry.Object = fd
		}
	}

	// append to Fields
	// Ensure AcroForm exists and has a default appearance (DA) so pdfcpu doesn't complain
	if o, found := ctx.RootDict.Find("AcroForm"); found {
		if form, ok := o.(types.Dict); ok {
			if ff, found := form.Find("Fields"); found {
				if arr, err := ctx.DereferenceArray(ff); err == nil {
					arr = append(arr, *fir)
					form["Fields"] = arr
				} else {
					form["Fields"] = types.Array{*fir}
				}
			} else {
				form["Fields"] = types.Array{*fir}
			}
			// set a minimal DA and DR if missing
			if _, ok := form.Find("DA"); !ok {
				form["DA"] = types.StringLiteral("/F1 12 Tf 0 g")
			}
			if _, ok := form.Find("DR"); !ok {
				form["DR"] = types.Dict{"Font": types.Dict{"F1": types.IndirectRef{ObjectNumber: 1}}}
			}
			ctx.RootDict["AcroForm"] = form
		} else {
			ctx.RootDict["AcroForm"] = types.Dict{"Fields": types.Array{*fir}, "DA": types.StringLiteral("/F1 12 Tf 0 g")}
		}
	} else {
		ctx.RootDict["AcroForm"] = types.Dict{"Fields": types.Array{*fir}, "DA": types.StringLiteral("/F1 12 Tf 0 g")}
	}

	// add widget to page Annots
	if ann, found := pageDict.Find("Annots"); found {
		if arr, err := ctx.DereferenceArray(ann); err == nil {
			arr = append(arr, *wir)
			pageDict["Annots"] = arr
		}
	} else {
		pageDict["Annots"] = types.Array{*wir}
	}

	return nil
}
