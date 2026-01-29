package newengine

import (
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// NewEngineRun applies two-layer time-limited protection: Fallback (default visible) and Normal (original content).
// Behavior:
// - Default: Fallback visible (OCG Fallback ON in OCProperties)
// - If viewer supports Acrobat JS and current time in [start,end]: show Normal (hide Fallback)
// - If viewer supports JS and now > end: keep Fallback and show alert expired
func NewEngineRun(ctx *model.Context, start, end time.Time, fallbackText string) error {
	fmt.Printf("Applying time-limited two-layer protection: start=%s end=%s\n", start, end)
	// register OCGs
	if err := injectOCGs(ctx); err != nil {
		return err
	}

	// For widget-only fallback: keep original page contents and create full-page fallback widgets
	for p := 1; p <= ctx.PageCount; p++ {
		if pageDict, _, _, err := ctx.PageDict(p, false); err == nil && pageDict != nil {
			if err := createFullPageFallbackWidget(ctx, p, fallbackText); err != nil {
				return err
			}
		}
	}

	// inject probe and JS that loops over fallback widgets
	if err := injectProbeAndJS(ctx, start, end, ctx.PageCount); err != nil {
		return err
	}

	return nil
}

// minimal OCG registration: create two OCGs and OCProperties with Fallback on by default
func injectOCGs(ctx *model.Context) error {
	oc1 := types.Dict{"Type": types.Name("OCG"), "Name": types.StringLiteral("Fallback")}
	oc2 := types.Dict{"Type": types.Name("OCG"), "Name": types.StringLiteral("Normal")}
	ir1, err := ctx.IndRefForNewObject(oc1)
	if err != nil {
		return err
	}
	ir2, err := ctx.IndRefForNewObject(oc2)
	if err != nil {
		return err
	}
	ocgs := types.Array{*ir1, *ir2}
	ocprops := types.Dict{
		"OCGs": ocgs,
		"D": types.Dict{
			"Order": ocgs,
			"ON":    types.Array{*ir1},
		},
	}
	ctx.RootDict["OCProperties"] = ocprops
	return nil
}

func injectJS(ctx *model.Context, start, end time.Time) error {
	// 计算时间窗口
	// build JS string with verbose debug alerts (formatted for debugging)
	jsText := fmt.Sprintf(`
try { app.alert("DEBUG: Current date: " + new Date().toUTCString()); } catch(e) {alert("DEBUG: app.alert not supported");}
try {
app.alert("DEBUG: Parsing start and end dates");
var startStr = "%s";
var endStr = "%s";
app.alert("DEBUG: StartStr: " + startStr + " EndStr: " + endStr);
var start = new Date(startStr);
var end = new Date(endStr);
var now = new Date();
var probe = false;
app.alert("DEBUG: Parsed Start date: " + start.toUTCString() + " End date: " + end.toUTCString());
}catch(e) { alert("DEBUG: error in initial try block: " + e); }
try { app.alert("DEBUG: Current date: " + now.toUTCString()); } catch(e) {app.alert("DEBUG: app.alert not supported");}

try {
  probe = !!this.getField('tag_probe');
  try { app.alert('DEBUG: probe present: ' + probe); } catch(e) {app.alert('DEBUG: probe check alert not supported');}
} catch(e) { try { app.alert('DEBUG: probe check threw'); } catch(e) {} }

try { app.alert('DEBUG: Start: ' + start.toUTCString() + ' End: ' + end.toUTCString()); } catch(e) {app.alert('DEBUG: app.alert not supported');}


try {var inRange = now >= start && now <= end; app.alert('DEBUG: inRange: ' + inRange); } catch(e) {app.alert('DEBUG: app.alert not supported');}

if(!probe) {
  try { app.alert('DEBUG: viewer does not support AcroForm/JS probe or probe missing'); } catch(e) {app.alert('DEBUG: app.alert not supported');}
}
app.alert('DEBUG: Executing layer switch logic');
if(probe) {
app.alert('DEBUG: Viewer supports AcroForm/JS probe');
  if(inRange) {
    try {
	app.alert('DEBUG: Now in valid range between ' + start.toUTCString() + ' and ' + end.toUTCString());
      this.getField('_FG_Normal').display = display.visible;
      this.getField('_FG_Fallback').display = display.hidden;
      try { app.alert('DEBUG: Showing Normal layer (original content)'); } catch(e) {}
    } catch(e) {
      try { app.alert('DEBUG: Failed to switch layers: ' + e); } catch(e) {}
    }
  } else if(now > end) {
    try { app.alert('Document expired on ' + end.toUTCString()); } catch(e) {}
  }
}
`, start.Format(time.RFC3339), end.Format(time.RFC3339))

	sd, err := ctx.NewStreamDictForBuf([]byte(jsText))
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
	// set as OpenAction
	act := types.Dict{"S": types.Name("JavaScript"), "JS": *ir}
	actIr, err := ctx.IndRefForNewObject(act)
	if err != nil {
		return err
	}
	ctx.RootDict["OpenAction"] = *actIr
	return nil
}
