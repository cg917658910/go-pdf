package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// injectJS injects an OpenAction JS that toggles form field values/visibility.
func injectJS(ctx *model.Context, start, end time.Time, fallbackText, expiredText string) {
	js := fmt.Sprintf(`(function(){
 try{
  var now = new Date();
  app.alert("Current date: " + now.toUTCString());
  // feature detection
  if(typeof this.getField !== 'function') return;
  try{ if(!this.getField('tag_probe')) return; }catch(e){ return; }

  var start=new Date("%s");
  var end=new Date("%s");
  app.alert("Start date: " + start.toUTCString() + "\nEnd date: " + end.toUTCString());
  try{
    if(now >= start && now <= end){
      try{ this.getField("_FG_Normal").display = display.visible; }catch(e){}
      try{ this.getField("_FG_Fallback").display = display.hidden; }catch(e){}
    } else if(now > end) {
      try{ app.alert("Document expired on " + end.toUTCString()); }catch(e){}
      try{ this.getField("_FG_Normal").display = display.hidden; }catch(e){}
      try{ this.getField("_FG_Fallback").display = display.visible; }catch(e){}
    } else {
      try{ this.getField("_FG_Normal").display = display.hidden; }catch(e){}
      try{ this.getField("_FG_Fallback").display = display.visible; }catch(e){}
    }
  }catch(e){}
 }catch(e){}
})();`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
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
