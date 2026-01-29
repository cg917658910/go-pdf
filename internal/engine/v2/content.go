package engine

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func extractPageContentAsXObject(
	ctx *model.Context,
	page types.Dict,
	pageNr int,
	ocg types.IndirectRef,
) (types.IndirectRef, error) {

	contents := page["Contents"]
	if contents == nil {
		return types.IndirectRef{}, fmt.Errorf("page has no contents")
	}

	// 内容流可能是一个或多个
	var streams types.Array
	switch c := contents.(type) {
	case types.IndirectRef:
		streams = types.Array{c}
	case types.Array:
		streams = c
	default:
		return types.IndirectRef{}, fmt.Errorf("unsupported contents type")
	}
	_, _, inhPAttrs, err := ctx.PageDict(pageNr, true)
	if err != nil {
		return types.IndirectRef{}, err
	}
	// 创建 Form XObject
	form := types.Dict{
		"Type":      types.Name("XObject"),
		"Subtype":   types.Name("Form"),
		"OC":        ocg, // ⭐ 关键：绑定 OCG_Normal
		"Resources": page["Resources"],
		"BBox":      inhPAttrs.MediaBox.Array(),
	}

	// 合并内容流
	form["Contents"] = streams

	formRef, err := ctx.IndRefForNewObject(form)

	// 将 Page 内容替换为仅绘制 fallback
	// 并在 Page Contents 末尾追加一个流：
	// q
	// /OC /OCG_Normal BDC
	// /NormalContent Do
	// EMC
	// Q
	// 这样 OCG 控制是否执行 Do，页面内容保持被 OCG 包裹但不被 JS 直接修改。

	return *formRef, err
}

func attachXObjectToPage(
	page types.Dict,
	xobj types.IndirectRef,
	ocg types.IndirectRef,
) {

	res := page["Resources"].(types.Dict)

	xobjs, ok := res["XObject"].(types.Dict)
	if !ok {
		xobjs = types.Dict{}
		res["XObject"] = xobjs
	}

	xobjs["NormalContent"] = xobj

	// Ensure OCG reference is available in page resource Properties mapping
	if props, ok := res["Properties"].(types.Dict); ok {
		props["OCG_Normal"] = ocg
		res["Properties"] = props
	} else {
		res["Properties"] = types.Dict{"OCG_Normal": ocg}
	}
}

func setFallbackContent(ctx *model.Context, page types.Dict) error {
	content := `
q
1 1 1 rg
0 0 10000 10000 re
f

0 0 0 rg
BT
/F1 24 Tf
200 500 Td
(FILE LOCKED - USE SUPPORTED PDF READER) Tj
ET
Q
`
	sd, err := ctx.NewStreamDictForBuf([]byte(content))
	if err != nil {
		return err
	}
	if err := sd.Encode(); err != nil {
		return err
	}
	ref, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	page["Contents"] = *ref
	return nil
}

func createUnlockWidget(
	ctx *model.Context,
	page types.Dict,
	pageNr int,
	normalXObj types.IndirectRef,
) error {

	// --- AP stream ---
	apContent := `
q
/NormalContent Do
Q
`
	// Compute inherited MediaBox early to set proper BBox in AP
	_, _, inhPAttrs, err := ctx.PageDict(pageNr, true)
	if err != nil {
		return err
	}

	apStream, err := ctx.NewStreamDictForBuf([]byte(apContent))
	if err != nil {
		return err
	}

	apStream.Dict = types.Dict{
		"Type":    types.Name("XObject"),
		"Subtype": types.Name("Form"),
		"BBox":    inhPAttrs.MediaBox.Array(),
		"Resources": types.Dict{
			"XObject": types.Dict{
				"NormalContent": normalXObj,
			},
		},
	}

	if err := apStream.Encode(); err != nil {
		return err
	}

	apRef, err := ctx.IndRefForNewObject(*apStream)
	if err != nil {
		return err
	}
	// --- Widget ---
	widget := types.Dict{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),

		"FT": types.Name("Btn"),
		"T":  types.StringLiteral("tag_unlock"),

		// 覆盖整页，但默认隐藏
		"Rect": inhPAttrs.MediaBox.Array(),

		"F": types.Integer(
			//(1 << 1) | // Hidden
			(1 << 2) | // Print
				(1 << 7), // Locked
		),

		"AP": types.Dict{
			"N": *apRef,
		},
	}

	ref, err := ctx.IndRefForNewObject(widget)

	if page["Annots"] == nil {
		page["Annots"] = types.Array{*ref}
	} else {
		page["Annots"] = append(page["Annots"].(types.Array), *ref)
	}

	return nil
}
