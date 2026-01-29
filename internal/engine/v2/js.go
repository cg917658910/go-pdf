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
  app.alert("JS activated");
  app.alert(typeof this.getAnnots!=="function");
 if(typeof this.getAnnots!=="function"){ app.alert("Annots API missing or JS blocked"); return; }
 app.alert("Annots API available");
 var now=new Date();
 var start=new Date("%s");
 var end=new Date("%s");
 var inRange = (now>=start && now<=end);
 app.alert("Current date: " + now.toUTCString());
 for(var p=0;p<this.numPages;p++){
  var ann = this.getAnnots(p);
  if(!ann) continue;
  for(var i=0;i<ann.length;i++){
    var a = ann[i];
    try{
      if(a.name && a.name=="tag_fallback"){
        if(inRange){
          a.display = display.hidden;
        } else {
          app.alert("expired");
          return;
        }
      }
    }catch(e){app.alert("error processing annot: " + e);}
  }
 }
}catch(e){app.alert("JS error: " + e);}
})();`,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
	)

	// Also register JS under Names/JavaScript to increase execution chance across readers.
	namesJS := types.Dict{"Names": types.Array{types.StringLiteral("OpenActionJS"), types.StringLiteral(js)}}
	if namesObj, found := ctx.RootDict.Find("Names"); found {
		if namesDict, ok := namesObj.(types.Dict); ok {
			namesDict["JavaScript"] = namesJS
			ctx.RootDict["Names"] = namesDict
		} else {
			ctx.RootDict["Names"] = types.Dict{"JavaScript": namesJS}
		}
	} else {
		ctx.RootDict["Names"] = types.Dict{"JavaScript": namesJS}
	}

	ctx.RootDict["OpenAction"] = types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	}
}

func injectTimeJS(ctx *model.Context, start, end time.Time) {
	js := fmt.Sprintf(`(function () {
  try {
    var start = new Date("%s");
    var end   = new Date("%s");
    var now   = new Date();

    // Toggle AcroForm field visibility if available
    if (typeof this.getField === "function") {
      var f = this.getField("tag_unlock");
      if (f) {
        try {
          if (now >= start && now <= end) {
            f.display = display.visible;
          } else {
		   app.alert("Document expired on " + end.toUTCString());
            f.display = display.hidden;
          }
        } catch (e) { }
      }
    }
  } catch (e) {app.alert("JS error: " + e);}
})();`, start.Format(time.RFC3339), end.Format(time.RFC3339))

	iref, err := ctx.IndRefForNewObject(types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.StringLiteral(js),
	})
	if err != nil {
		fmt.Printf("injectTimeJS: %v\n", err)
	}
	ctx.RootDict["Names"] = types.Dict{
		"JavaScript": types.Dict{
			"Names": types.Array{
				types.StringLiteral("docjs"),
				*iref,
			},
		},
	}
}
