package engine

import (
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func injectOpenActionJS(ctx *model.Context, start, end time.Time) {

	js := fmt.Sprintf(`(function(){
try{
 if(typeof this.getField!=="function") return;

 var f=this.getField("tag_fallback");
 if(!f) return;

 var now=new Date();
 var start=new Date("%s");
 var end=new Date("%s");

 if(now<start||now>end){
  app.alert("expired");
  return;
 }

 f.display=display.hidden;
}catch(e){}
})();`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)

	ctx.RootDict["OpenAction"] = types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	}
}
