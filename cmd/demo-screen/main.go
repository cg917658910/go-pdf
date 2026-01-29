package main

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func main() {

	in := "./cg.pdf"
	out := "./cg-screen.pdf"

	//conf := model.NewDefaultConfiguration()

	ctx, err := api.ReadContextFile(in)
	if err != nil {
		panic(err)
	}

	for p := 1; p <= ctx.PageCount; p++ {
		pageDict, _, _, err := ctx.PageDict(p, false)
		if err != nil {
			fmt.Printf("get page dict for page %d: %v\n", p, err)
			panic(err)
		}
		if pageDict == nil {
			continue
		}

		err = addScreen(ctx, pageDict, p)
		if err != nil {
			panic(err)
		}
	}

	err = api.WriteContextFile(ctx, out)
	if err != nil {
		panic(err)
	}

	fmt.Println("done:", out)
}

func addScreen(ctx *model.Context, page types.Dict, p int) error {
	//mb := page.MediaBox()
	_, _, inhPAttrs, err := ctx.PageDict(p, true)
	if err != nil {
		return err
	}
	mb := inhPAttrs.MediaBox
	rect := types.Array{
		types.Float(mb.LL.X),
		types.Float(mb.LL.Y),
		types.Float(mb.UR.X),
		types.Float(mb.UR.Y),
	}

	// --- AP (Form XObject) ---
	ap := screenAppearance(ctx)
	if err := ap.Encode(); err != nil {
		panic(err)
	}
	apRef, err := ctx.IndRefForNewObject(ap)
	if err != nil {
		panic(err)
	}
	screen := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Screen"),

		"Rect": rect,

		// 锁死交互
		"F": types.Integer(
			(1 << 2) | // Print
				(1 << 3) | // NoZoom
				(1 << 4) | // NoRotate
				(1 << 7), // Locked
		),

		"AP": types.Dict{
			"N": *apRef,
		},
	}

	ref, err := ctx.IndRefForNewObject(screen)
	if err != nil {
		panic(err)
	}

	if page["Annots"] == nil {
		page["Annots"] = types.Array{*ref}
	} else {
		page["Annots"] = append(page["Annots"].(types.Array), *ref)
	}

	return nil
}

func screenAppearance(ctx *model.Context) *types.StreamDict {
	content := `
q
1 1 1 rg
0 0 10000 10000 re
f

0 0 0 rg
BT
/F1 36 Tf
200 500 Td
(SCREEN FALLBACK TEST) Tj
ET
Q
`

	sd, err := ctx.NewStreamDictForBuf([]byte(content))
	if err != nil {
		panic(err)
	}

	sd.Dict = types.Dict{
		"Type":    types.Name("XObject"),
		"Subtype": types.Name("Form"),
		"BBox": types.Array{
			types.Float(0),
			types.Float(0),
			types.Float(10000),
			types.Float(10000),
		},
		"Resources": types.Dict{
			"Font": types.Dict{
				"F1": types.Dict{
					"Type":     types.Name("Font"),
					"Subtype":  types.Name("Type1"),
					"BaseFont": types.Name("Helvetica"),
				},
			},
		},
	}
	if err := sd.Encode(); err != nil {
		panic(err)
	}
	return sd
}
