package engine

import (
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func addFallbackWidget(
	ctx *model.Context,
	page types.Dict,
	p int,
	ocg types.IndirectRef,
	text string,
) error {

	_, _, inhPAttrs, err := ctx.PageDict(p, true)
	if err != nil {
		return err
	}
	mediaBox := inhPAttrs.MediaBox
	rect := types.Array{
		types.Float(mediaBox.LL.X),
		types.Float(mediaBox.LL.Y),
		types.Float(mediaBox.UR.X),
		types.Float(mediaBox.UR.Y),
	}

	// 外观流（AP）
	appearance := fallbackAppearance(text)

	apStream, err := ctx.NewStreamDictForBuf(appearance)
	if err != nil {
		return err
	}

	apRef, err := ctx.IndRefForNewObject(apStream)
	if err != nil {
		return err
	}

	widget := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"FT":      types.Name("Tx"),
		"T":       types.StringLiteral("tag_fallback"),
		"Rect":    rect,
		"F":       types.Integer(4),
		"AP": types.Dict{
			"N": *apRef,
		},
		"OC": ocg, // ⭐关键：Widget 归属 Fallback OCG
	}

	annotRef, err := ctx.IndRefForNewObject(widget)
	if err != nil {
		return err
	}

	annots := page["Annots"]
	if annots == nil {
		page["Annots"] = types.Array{*annotRef}
	} else {
		page["Annots"] = append(annots.(types.Array), *annotRef)
	}

	return nil
}

func fallbackAppearance(text string) []byte {
	return []byte(fmt.Sprintf(`
q
0.9 g
0 0 10000 10000 re
f
0 g
BT
/F1 36 Tf
100 500 Td
(%s) Tj
ET
Q
`, escape(text)))
}
func escape(s string) string {
	// 最小安全转义
	return strings.ReplaceAll(s, "(", "\\(")
}
