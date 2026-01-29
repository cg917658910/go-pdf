package engine

import (
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func injectJS(ctx *model.Context, start, end time.Time) {
	js := fmt.Sprintf(`(function(){
try{
 if(typeof this.getField!=="function")throw 1;
 if(!this.getField("tag_probe"))throw 2;
 if(typeof this.setOCGState!=="function")throw 3;

 var now=new Date();
 var start=new Date("%s");
 var end=new Date("%s");

 this.setOCGState({cState:["OFF"],oCGs:["OCG_Fallback"]});

 if(now>=start&&now<=end){
  this.setOCGState({cState:["ON"],oCGs:["OCG_Normal"]});
 }else{
  this.setOCGState({cState:["ON"],oCGs:["OCG_Expired"]});
 }
}catch(e){
 this.setOCGState({cState:["ON"],oCGs:["OCG_Fallback"]});
}})();`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)

	ctx.RootDict["OpenAction"] = types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	}
}
