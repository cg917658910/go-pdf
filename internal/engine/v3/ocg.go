package engine

import (
	"bytes"
	"fmt"
	"log"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func createOCG(ctx *model.Context, name string) *types.IndirectRef {
	d := types.Dict{
		"Type": types.Name("OCG"),
		"Name": types.StringLiteral(name),
	}
	ref, err := ctx.IndRefForNewObject(d)
	if err != nil {
		log.Printf("Error creating OCG %s: %v\n", name, err)
	}
	return ref
}
func applyOCProperties(
	ctx *model.Context,
	ocgs []*types.IndirectRef,
) {
	arr := types.Array{}
	for _, ref := range ocgs {
		arr = append(arr, *ref)
	}

	ctx.RootDict["OCProperties"] = types.Dict{
		"OCGs": arr,
		"D": types.Dict{
			"ON": arr, // 🔥 默认全部 ON
		},
	}
}

func buildMaskXObject(
	ctx *model.Context,
	mediaBox types.Array,
) (*types.IndirectRef, error) {

	w := mediaBox[2]
	h := mediaBox[3]

	content := fmt.Sprintf(`
q
1 g
0 0 %v %v re
f
Q
`, w, h)

	sd, err := ctx.NewStreamDictForBuf(
		[]byte(content),
	)
	if err != nil {
		return nil, err
	}
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	sd.Dict["Type"] = types.Name("XObject")
	sd.Dict["Subtype"] = types.Name("Form")
	sd.Dict["BBox"] = mediaBox

	return ctx.IndRefForNewObject(*sd)
}

func buildTextXObject(
	ctx *model.Context,
	mediaBox types.Array,
	text string,
) (*types.IndirectRef, error) {

	h := mediaBox[3]

	content := fmt.Sprintf(`
q
0 g
BT
/F1 28 Tf
72 %v Td
(%s) Tj
ET
Q
`, h.(types.Float)/2, types.StringLiteral(text))

	sd, err := ctx.NewStreamDictForBuf(
		[]byte(content),
	)
	if err != nil {
		return nil, err
	}
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	sd.Dict["Type"] = types.Name("XObject")
	sd.Dict["Subtype"] = types.Name("Form")
	sd.Dict["BBox"] = mediaBox

	return ctx.IndRefForNewObject(*sd)
}

func rewritePageWithMasks(
	ctx *model.Context,
	pageDict types.Dict,
	masks []*types.IndirectRef,
	text *types.IndirectRef,
	maskOCGs []*types.IndirectRef,
	textOCG *types.IndirectRef,
) error {

	var buf bytes.Buffer

	//buf.WriteString("q\n/NormalContent Do\nQ\n")

	for i := range masks {
		buf.WriteString(fmt.Sprintf(
			"/OC %v BDC\n/Mask_%02d Do\nEMC\n",
			maskOCGs[i],
			i,
		))
	}

	buf.WriteString(fmt.Sprintf(
		"/OC %v BDC\n/Text_0 Do\nEMC\n",
		textOCG,
	))

	sd, err := ctx.NewStreamDictForBuf(
		buf.Bytes(),
	)
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
	// append to existing Contents which currently is a single IndRef from setFallbackContent
	if c := pageDict["Contents"]; c != nil {
		switch t := c.(type) {
		case types.IndirectRef:
			pageDict["Contents"] = types.Array{t, *ref}
		case types.Array:
			pageDict["Contents"] = append(t, *ref)
		default:
			return fmt.Errorf("unsupported Contents type when appending Do stream: %T", t)
		}
	} else {
		pageDict["Contents"] = *ref
	}
	//pageDict["Contents"] = *ref
	return nil
}

func injectOCGResources(
	ctx *model.Context,
	pageDict types.Dict,
	masks []*types.IndirectRef,
	text *types.IndirectRef,
	maskOCGs []*types.IndirectRef,
	textOCG *types.IndirectRef,
) {
	res := pageDict["Resources"].(types.Dict)

	xobj, ok := res["XObject"].(types.Dict)
	if !ok {
		xobj = types.Dict{}
		res["XObject"] = xobj
	}

	//xobj["NormalContent"] = *normal

	for i, m := range masks {
		xobj[fmt.Sprintf("Mask_%02d", i)] = *m
	}

	xobj["Text_0"] = *text

	props := types.Dict{}
	res["Properties"] = props

	for i, ocg := range maskOCGs {
		props[fmt.Sprintf("mask_0_%02d", i)] = *ocg
	}
	props["text_0"] = *textOCG

}
