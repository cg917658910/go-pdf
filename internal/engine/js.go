package engine

import (
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func injectJS(ctx *model.Context, start, end time.Time) {
	js := fmt.Sprintf(`(function(){\ntry{\n var now = new Date();\n // check for form/getField and setOCGState support without alerting\n var hasGetField = (typeof this.getField === "function");\n var hasSetOCG = (typeof this.setOCGState === "function");\n var hasTag = false;\n try{ hasTag = !!this.getField("tag_probe"); }catch(e){ hasTag = false; }\n // if tag or setOCG not supported, skip toggling\n if(!hasGetField || !hasTag || !hasSetOCG){ return; }\n\n var start=new Date("%s");\n var end=new Date("%s");\n\n try{\n  this.setOCGState({cState:["OFF"],oCGs:["OCG_Fallback"]});\n  if(now>=start&&now<=end){\n   this.setOCGState({cState:["ON"],oCGs:["OCG_Normal"]});\n  }else{\n   this.setOCGState({cState:["ON"],oCGs:["OCG_Expired"]});\n  }\n }catch(e){}\n}catch(e){}\n})();`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)

	ctx.RootDict["OpenAction"] = types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	}
}
