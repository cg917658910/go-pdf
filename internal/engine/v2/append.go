package engine

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// appendDoNormalContent appends a content stream to the page that executes the NormalContent XObject
// wrapped in an OCG BDC/EMC block so the OCG controls whether the Do is executed/visible.
func appendDoNormalContent(ctx *model.Context, page types.Dict) error {
	content := `
q
/OC /OCG_Normal BDC
/NormalContent Do
EMC
Q
`
// Note: the BDC operator references a name /OCG_Normal which must be defined in the page or resource
// name dictionary. We already add a Properties entry mapping "OCG_Normal" to the OCG indirect ref.	// create stream
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

	// append to existing Contents which currently is a single IndRef from setFallbackContent
	if c := page["Contents"]; c != nil {
		switch t := c.(type) {
		case types.IndirectRef:
			page["Contents"] = types.Array{t, *ref}
		case types.Array:
			page["Contents"] = append(t, *ref)
		default:
			return fmt.Errorf("unsupported Contents type when appending Do stream: %T", t)
		}
	} else {
		page["Contents"] = *ref
	}

	return nil
}
