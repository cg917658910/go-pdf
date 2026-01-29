package engine

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func readPageContent(ctx *model.Context, page types.Dict) ([]byte, error) {
	obj := page["Contents"]
	if obj == nil {
		return nil, nil
	}
	sd, _, err := ctx.DereferenceStreamDict(obj)
	if err != nil {
		return nil, err
	}
	return sd.Content, nil
}

func wrapOCG(name string, b []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "/OC /%s BDC\n", name)
	buf.Write(b)
	buf.WriteString("\nEMC\n")
	return buf.Bytes()
}

func rewritePageContent(
	ctx *model.Context,
	page types.Dict,
	normal, fallback, expired []byte,
) error {

	// Create a Form XObject from the original page content and register it.
	orig := normal
	if orig == nil {
		orig = []byte{}
	}

	// Build a Form XObject as a StreamDict (so the stream gets registered properly).
	fsd, err := ctx.NewStreamDictForBuf(orig)
	if err != nil {
		return err
	}
	// Ensure form identity and basic entries.
	fsd.Dict["Type"] = types.Name("XObject")
	fsd.Dict["Subtype"] = types.Name("Form")
	fsd.Dict["BBox"] = types.Array{types.Integer(0), types.Integer(0), types.Integer(612), types.Integer(792)}
	fsd.Dict["Resources"] = types.Dict{}

	if err := fsd.Encode(); err != nil {
		return err
	}

	// register form xobject as an indirect StreamDict object
	formIr, err := ctx.IndRefForNewObject(*fsd)
	if err != nil {
		return err
	}

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
