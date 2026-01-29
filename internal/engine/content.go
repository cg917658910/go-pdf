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

	final := bytes.Join([][]byte{
		wrapOCG("OCG_Fallback", fallback),
		wrapOCG("OCG_Normal", normal),
		wrapOCG("OCG_Expired", expired),
	}, []byte("\n"))

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

	// Debug: ensure the xref table entry for this new object is a value StreamDict
	if entry, found := ctx.FindTableEntryLight(int(ir.ObjectNumber)); found {
		// When entry.Object is a pointer to types.StreamDict that's an issue for pdfcpu writer.
		switch entry.Object.(type) {
		case *types.StreamDict:
			// Convert in-place to value type to be safe.
			entry.Object = *(entry.Object.(*types.StreamDict))
		default:
			// nothing
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
