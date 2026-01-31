package engine

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func injectOpenActionJS(ctx *model.Context, start, end time.Time, experiredText, unsupportedText string) {

	// quote strings to be safely embedded into JS code
	// Use QuoteToASCII so non-ASCII characters (例如中文) are escaped as \uXXXX,
	// 保证生成的 JS 代码只包含 ASCII，从而避免在 PDF 中出现编码/乱码问题。
	qExpired := strconv.QuoteToASCII(experiredText)
	qUnsupported := strconv.QuoteToASCII(unsupportedText)

	js := fmt.Sprintf(`(function(){
  try{
	app.alert("Js feature detection...");
    if(this.__ocg_js_executed) return; this.__ocg_js_executed = true;
    var start = new Date("%s");
    var end = new Date("%s");
    var now = new Date();
    var inRange = (now >= start && now <= end);
    var ocgName = "OCG_Normal";
    try {
      if(!inRange){
        app.alert(%s);
        return;  
      }
      if (typeof this.setOCGState === "function") {
        this.setOCGState(ocgName, inRange);
      } else if (typeof this.getOCGs === "function") {
        var ocgs = this.getOCGs();
		app.alert("Found " + ocgs.length + " OCGs.");
        for (var i = 0; i < ocgs.length; i++) {
          var o = ocgs[i];
          o.state = false;
        }
      }
    } catch (e) { app.alert(%s + ' ' + e); }
    } catch (e) { }
})();`, start.Format(time.RFC3339), end.Format(time.RFC3339), qExpired, qUnsupported)

	iref, err := ctx.IndRefForNewObject(types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	})
	if err != nil {
		fmt.Printf("injectOpenActionJS: %v\n", err)
	}

	ctx.RootDict["Names"] = types.Dict{
		"JavaScript": types.Dict{
			"Names": types.Array{
				types.StringLiteral("OpenActionJS"),
				*iref,
			},
		},
	}

	// Set OpenAction to the JavaScript action indirect reference.
	ctx.RootDict["OpenAction"] = *iref
}
