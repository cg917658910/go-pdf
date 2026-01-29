package newengine

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func injectProbeAndJS(ctx *model.Context, start, end time.Time, pageCount int) error {
	// create probe field
	var form types.Dict
	if o, found := ctx.RootDict.Find("AcroForm"); found {
		if d, ok := o.(types.Dict); ok {
			form = d
		}
	}
	if form == nil {
		form = types.Dict{}
		ctx.RootDict["AcroForm"] = form
	}

	probe := types.Dict{"T": types.StringLiteral("tag_probe"), "FT": types.Name("Tx")}
	probeIr, err := ctx.IndRefForNewObject(probe)
	if err != nil {
		return err
	}
	if o, found := form.Find("Fields"); found {
		if arr, err := ctx.DereferenceArray(o); err == nil {
			arr = append(arr, *probeIr)
			form["Fields"] = arr
		} else {
			form["Fields"] = types.Array{*probeIr}
		}
	} else {
		form["Fields"] = types.Array{*probeIr}
	}

	// Build JS to hide fallback fields for each page when in range, or alert if expired
	js := bytes.NewBufferString(`
	try{
	app.alert('JS activated');
	}catch(e){
	 app.alert('no app');
	}
	\n`)
	fmt.Fprintf(js, "var start = new Date('%s');\nvar end = new Date('%s');\nvar now = new Date();\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
	fmt.Fprintf(js, "try{ if(!this.getField('tag_probe')) return; }catch(e){ app.alert('not tag_p')return; }\n")
	fmt.Fprintf(js, "var pages = [")
	for i := 1; i <= pageCount; i++ {
		if i > 1 {
			fmt.Fprintf(js, ",")
		}
		fmt.Fprintf(js, "'fallback_p%d'", i)
	}
	fmt.Fprintf(js, "]\n")
	fmt.Fprintf(js, "if(now >= start && now <= end){ for(var i=0;i<pages.length;i++){ try{ this.getField(pages[i]).display = display.hidden; }catch(e){} } }else if(now > end){ try{ app.alert('Document expired'); }catch(e){} }\n")

	sd, err := ctx.NewStreamDictForBuf(js.Bytes())
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
	act := types.Dict{"S": types.Name("JavaScript"), "JS": *ir}
	actIr, err := ctx.IndRefForNewObject(act)
	if err != nil {
		return err
	}
	// Also register the JS in Names under JavaScript (helps some readers find/execute JS)
	// Build Names dictionary entry
	if namesObj, found := ctx.RootDict.Find("Names"); found {
		if namesDict, ok := namesObj.(types.Dict); ok {
			namesDict["JavaScript"] = types.Dict{"Names": types.Array{types.StringLiteral("OpenActionJS"), *ir}}
			ctx.RootDict["Names"] = namesDict
		} else {
			ctx.RootDict["Names"] = types.Dict{"JavaScript": types.Dict{"Names": types.Array{types.StringLiteral("OpenActionJS"), *ir}}}
		}
	} else {
		ctx.RootDict["Names"] = types.Dict{"JavaScript": types.Dict{"Names": types.Array{types.StringLiteral("OpenActionJS"), *ir}}}
	}

	ctx.RootDict["OpenAction"] = *actIr
	ctx.RootDict["AcroForm"] = form
	return nil
}
