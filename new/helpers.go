package newengine

import (
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func rewritePageContent(
	ctx *model.Context,
	page types.Dict,
	normal, fallback, expired []byte,
) error {
	// Create a Form XObject from the original page content and register it.
	orig := normal
	if orig == nil {
		fmt.Println("No original content found for page; using empty content for Form XObject")
		orig = []byte{}
	}
	fmt.Printf("Original page content length: %d bytes\n", len(orig))
	// Build a basic Form XObject stream dict
	fd := types.Dict{
		"Type":      types.Name("XObject"),
		"Subtype":   types.Name("Form"),
		"BBox":      types.Array{types.Integer(0), types.Integer(0), types.Integer(612), types.Integer(792)},
		"Resources": types.Dict{},
	}

	// content stream for the form is the original page content
	fsd, err := ctx.NewStreamDictForBuf(orig)
	if err != nil {
		return err
	}
	if err := fsd.Encode(); err != nil {
		return err
	}

	// If possible, copy page Resources into the Form XObject Resources so the form can resolve fonts/images
	if pdRes, _ := ctx.DereferenceDict(page["Resources"]); pdRes != nil {
		fd["Resources"] = pdRes
	} else if fd["Resources"] == nil {
		fd["Resources"] = types.Dict{}
	}

	// Set Form BBox from page MediaBox if available
	if mb, found := page["MediaBox"]; found {
		if arr, err := ctx.DereferenceArray(mb); err == nil && len(arr) == 4 {
			fd["BBox"] = types.Array{arr[0], arr[1], arr[2], arr[3]}
		}
	}

	// Merge the stream dict entries (Length/Filter/Decoded) but keep Resources separate.
	for k, v := range fsd.Dict {
		if k == "Resources" {
			continue
		}
		fd[k] = v
	}

	// register form xobject
	formIr, err := ctx.IndRefForNewObject(fd)
	if err != nil {
		return err
	}

	// Add the form XObject into the page's Resources.XObject under a predictable name
	res, _ := ctx.DereferenceDict(page["Resources"])
	if res == nil {
		res = types.Dict{}
		page["Resources"] = res
	}
	xobj, _ := ctx.DereferenceDict(res["XObject"])
	if xobj == nil {
		xobj = types.Dict{}
		res["XObject"] = xobj
	}
	// use a unique name derived from the form object's number
	name := fmt.Sprintf("NX%d", formIr.ObjectNumber)
	xobj[name] = *formIr
	res["XObject"] = xobj
	page["Resources"] = res

	// Replace page contents with Fallback static text stream
	fallbackStream := textBlock(string(fallback))
	sd, err := ctx.NewStreamDictForBuf(fallbackStream)
	if err != nil {
		return err
	}
	if err := sd.Encode(); err != nil {
		return err
	}
	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}
	page["Contents"] = *ir

	// Expose the Form XObject via AP on the _FG_Normal widget by setting its appearance in the widget creation step.
	// We store the form indirect ref in the page under a custom key so injectTagField can pick it up.
	page["_NormalFormXObject"] = *formIr

	// Also create a simple appearance stream that references the XObject via Do operator so some viewers render it as widget appearance
	// Build AP stream content: q /NX{obj#} Do Q
	apContent := fmt.Sprintf("q /%s Do Q", fmt.Sprintf("NX%d", formIr.ObjectNumber))
	sdap, err := ctx.NewStreamDictForBuf([]byte(apContent))
	if err == nil {
		_ = sdap.Encode()
		if api, err := ctx.IndRefForNewObject(*sdap); err == nil {
			page["_NormalAP"] = *api
		}
	}

	return nil
}

func textBlock(text string) []byte {
	return []byte(fmt.Sprintf(`
BT
/F1 24 Tf
72 400 Td
(%s) Tj
ET
`, escape(text)))
}

func escape(s string) string {
	// 最小安全转义
	return strings.ReplaceAll(s, "(", "\\(")
}

func injectTagField(ctx *model.Context, fallbackText, expiredText string) error {
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

	// For every page create two invisible text fields (Fallback/Normal) with same position.
	for p := 1; p <= ctx.PageCount; p++ {
		pageDict, _, _, err := ctx.PageDict(p, false)
		if err != nil || pageDict == nil {
			continue
		}

		// choose a rect covering the full page so the Normal appearance overlays the entire page
		rect := types.Array{types.Integer(0), types.Integer(0), types.Integer(612), types.Integer(792)}

		names := []string{"_FG_Fallback", "_FG_Normal"}

		// Create fields and widgets
		for _, nm := range names {
			fmt.Printf("Creating field %s on page %d\n", nm, p)
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
			// Attach AP Normal form if available on page
			if nm == "_FG_Normal" {
				if fx, ok := pageDict["_NormalFormXObject"]; ok {
					// set appearance dictionary N to reference this XObject
					if ref, ok := fx.(types.IndirectRef); ok {
						// Instead of embedding the form directly in AP, reference the XObject from the page Resources
						// The form was registered under name NX{formIr.ObjectNumber} in page.Resources.XObject
						// AP.N should be a stream XObject reference; here set AP.N to directly reference the Form XObject stream
						ap := types.Dict{"N": ref}
						fmt.Printf("Setting AP for widget %s on page %d to reference XObject %d\n", nm, p, ref.ObjectNumber)
						if entry, found := ctx.FindTableEntryLight(int(wir.ObjectNumber)); found && entry != nil && entry.Object != nil {
							if wd, ok := entry.Object.(types.Dict); ok {
								wd["AP"] = ap
								wd["F"] = types.Integer(4)
								entry.Object = wd
								fmt.Printf("AP set object wd=%v\n", wd["type"])
								fmt.Printf("AP set for widget %s on page %d\n", nm, p)
							}
						}
					}
				} else {
					fmt.Printf("No _NormalFormXObject found on page %d for widget %s\n", p, nm)
				}
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
