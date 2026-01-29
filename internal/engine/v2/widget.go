package engine

import (
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	AnnotPrint          = 1 << 2 // 4
	AnnotNoZoom         = 1 << 3 // 8
	AnnotNoRotate       = 1 << 4 // 16
	AnnotLocked         = 1 << 7 // 128
	AnnotLockedContents = 1 << 9 // 512
)

const (
	FieldReadOnly = 1 << 0
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
	//w := mediaBox.UR.X - mediaBox.LL.X
	//h := mediaBox.UR.Y - mediaBox.LL.Y
	rect := types.Array{
		types.Float(mediaBox.LL.X),
		types.Float(mediaBox.LL.Y),
		types.Float(mediaBox.UR.X),
		types.Float(mediaBox.UR.Y),
	}

	// 外观流（AP）
	//appearance := fallbackAppearance(ctx,text, w, h)
	ap := fallbackAppearance(ctx, "文件显示错误！请使用 Adobe Reader 或福昕 PDF 阅读器打开。")

	// Ensure AP is a Form XObject with correct BBox so it covers the page.
	/* apStream.Dict["Type"] = types.Name("XObject")
	apStream.Dict["Subtype"] = types.Name("Form")
	apStream.Dict["BBox"] = types.Array{types.Float(0), types.Float(0), types.Float(w), types.Float(h)}
	apStream.Dict["Resources"] = types.Dict{}

	if err := apStream.Encode(); err != nil {
		return err
	} */

	apRef, err := ctx.IndRefForNewObject(*ap)
	if err != nil {
		return err
	}

	widget := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Screen"),
		"NM":      types.StringLiteral("tag_fallback"),
		"Rect":    rect,
		"F":       types.Integer(4),
		"AP": types.Dict{
			"N": *apRef,
		},
		"OC": ocg, // ⭐关键：Annotation 归属 Fallback OCG
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

func fallbackAppearance(ctx *model.Context, text string) *types.StreamDict {

	content := fmt.Sprintf(`
q
%% 绝对不透明背景
1 1 1 rg
0 0 10000 10000 re
f

%% 文本
0 0 0 rg
BT
/F1 36 Tf
100 500 Td
(%s) Tj
ET
Q
`, escape(text))

	sd, err := ctx.NewStreamDictForBuf([]byte(content))
	if err != nil {
		fmt.Printf("fallbackAppearance: %v\n", err)
		return nil
	}

	// ⭐ 核心：明确这是 Form XObject
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
		fmt.Printf("fallbackAppearance encode: %v\n", err)
		return nil
	}

	return sd
}

func escape(s string) string {
	// 最小安全转义
	return strings.ReplaceAll(s, "(", "\\(")
}
