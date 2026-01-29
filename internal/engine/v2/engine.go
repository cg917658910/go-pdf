package engine

import (
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type Options struct {
	Input     string
	Output    string
	StartTime time.Time
	EndTime   time.Time
	Watermark string

	DisablePrint bool
	DisableCopy  bool
}

func Run(opt Options) error {
	//conf := model.NewDefaultConfiguration()
	fmt.Printf("Applying time-limited two-layer protection from %s to %s\n", opt.StartTime.Format(time.RFC3339), opt.EndTime.Format(time.RFC3339))
	ctx, err := api.ReadContextFile(opt.Input)
	if err != nil {
		fmt.Printf("read context file: %v\n", err)
		return err
	}
	/* ctx.RootDict["AcroForm"] = types.Dict{
		"NeedAppearances": types.Boolean(false),
	} */
	// 1. 创建 OCG
	normalOCG, _ := ensureOCGs(ctx)

	// 2. 每一页加 Fallback Widget（遮罩）
	// 2. Pages
	for p := 1; p <= ctx.PageCount; p++ {
		pageDict, _, _, err := ctx.PageDict(p, true)
		if err != nil {
			fmt.Printf("get page dict for page %d: %v\n", p, err)
			return err
		}
		if pageDict == nil {
			continue
		}
		// 1.把原页面内容封装成 Form XObject
		pxd, err := extractPageContentAsXObject(ctx, pageDict, p, normalOCG)
		if err != nil {
			fmt.Printf("extract page content as XObject for page %d: %v\n", p, err)
			return err
		}
		// 2.重写页面 Contents（只画 fallback）
		err = setFallbackContent(ctx, pageDict)
		if err != nil {
			fmt.Printf("set fallback content for page %d: %v\n", p, err)
			return err
		}
		// 3.把 NormalContent XObject 加回页面 Resources
		attachXObjectToPage(pageDict, pxd)
		// 4.创建隐藏 Widget，AP 里 Do NormalContent
		createUnlockWidget(ctx, pageDict, p, pxd)
		/* if err := addFallbackWidget(ctx, pageDict, p, fallbackOCG, opt.Watermark); err != nil {
			fmt.Printf("add fallback widget to page %d: %v\n", p, err)
			return err
		} */
	}

	// 3. 注入 JS（只隐藏 Widget）
	//injectOpenActionJS(ctx, opt.StartTime, opt.EndTime)
	injectTimeJS(ctx, opt.StartTime, opt.EndTime)
	// 4. 权限限制
	if opt.DisablePrint || opt.DisableCopy {
		//restrictPermissions(ctx, opt.DisablePrint, opt.DisableCopy)
	}
	fmt.Println("Applying time-limited two-layer protection completed.")
	return api.WriteContextFile(ctx, opt.Output)
}
