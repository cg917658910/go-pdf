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

	// Keep existing page contents as the Normal layer base
	normalLayer := normal
	if normalLayer == nil {
		normalLayer = []byte{}
	}

	// Create three text widget fields per page: Fallback, Normal, Expired.
	// We'll leave the Normal field visible by default, and set Expired/Default based on time.
	// Build a combined static content stream that draws nothing; actual display happens via form widgets.
	final := normalLayer

	sd, err := ctx.NewStreamDictForBuf(final)
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
