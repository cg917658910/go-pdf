package engine

import (
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// injectJS injects an OpenAction JS that toggles form field values/visibility.
func injectJS(ctx *model.Context, start, end time.Time, fallbackText, expiredText string) {
	js := fmt.Sprintf(`(function(){
 try{
  var now = new Date();
  var hasGetField = (typeof this.getField === "function");
  var hasTag = false;
  try{ hasTag = !!this.getField("tag_probe"); }catch(e){ hasTag = false; }
  if(!hasGetField || !hasTag){ return; }

  var start=new Date("%s");
  var end=new Date("%s");

  try{
    if(now>=start&&now<=end){
      try{ this.getField("_FG_Normal").value = ""; }catch(e){}
      try{ this.getField("_FG_Fallback").display = display.hidden; }catch(e){}
      try{ this.getField("_FG_Expired").display = display.hidden; }catch(e){}
    }else{
      try{ this.getField("_FG_Expired").value = "%s"; }catch(e){}
      try{ this.getField("_FG_Normal").display = display.hidden; }catch(e){}
      try{ this.getField("_FG_Fallback").display = display.hidden; }catch(e){}
    }
  }catch(e){}
 }catch(e){}
})();`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		escapeJSString(expiredText),
	)

	jact := types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	}
	ir, _ := ctx.IndRefForNewObject(jact)
	if ir != nil {
		ctx.RootDict["OpenAction"] = *ir
	} else {
		ctx.RootDict["OpenAction"] = jact
	}
}

// escapeJSString escapes backslashes and quotes for embedding in JS string literal.
func escapeJSString(s string) string {
	// minimal escaping
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
