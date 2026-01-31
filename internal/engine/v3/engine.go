package engine

import (
	"fmt"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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
	ctx, err := api.ReadContextFile(opt.Input)
	if err != nil {
		fmt.Printf("read context file: %v\n", err)
		return err
	}
	const maskNum = 3
	// 1. 创建 OCG
	//normalOCG := createOCG(ctx,"Normal_OCG")
	fallbackOCG := createOCG(ctx, "text_0")
	maskOCGs := make([]*types.IndirectRef, maskNum)
	allOCGs := make([]*types.IndirectRef, 0, maskNum+1)
	for i := 0; i < maskNum; i++ {
		maskOCGs[i] = createOCG(ctx, fmt.Sprintf("mask_0_%02d", i+1))
	}
	//allOCGs = append(allOCGs, normalOCG)
	allOCGs = append(allOCGs, maskOCGs...)
	allOCGs = append(allOCGs, fallbackOCG)

	applyOCProperties(ctx, allOCGs)

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
		// 提取页面内容
		normalXObj, err := extractPageContentAsXObject(ctx, pageDict, p)
		if err != nil {
			fmt.Printf("extract page content as XObject for page %d: %v\n", p, err)
			return err
		}
		// 绑定 OCG 资源
		_, _, inhPAttrs, err := ctx.PageDict(p, false)
		if err != nil {
			fmt.Printf("get inherited page attrs for page %d: %v\n", p, err)
			return err
		}
		mediaBox := inhPAttrs.MediaBox.Array()
		maskXObjs := make([]*types.IndirectRef, maskNum)
		for i := 0; i < maskNum; i++ {
			mxobj, err := buildMaskXObject(ctx, mediaBox)
			if err != nil {
				fmt.Printf("build mask xobject for page %d: %v\n", p, err)
				return err
			}
			maskXObjs[i] = mxobj
		}
		fallbackXObj, err := buildTextXObject(ctx, mediaBox, opt.Watermark)
		injectOCGResources(ctx, pageDict, normalXObj, maskXObjs, fallbackXObj, maskOCGs, fallbackOCG)
		// 3. 重写页面内容
		rewritePageWithMasks(ctx, pageDict, maskXObjs, fallbackXObj, maskOCGs, fallbackOCG)
	}
	injectOpenActionJS(ctx, opt.StartTime, opt.EndTime, "已过期", "不支持的查看器")

	fmt.Println("Applying time-limited two-layer protection completed.")
	// 5. 写出文件
	// 判断输出文件是否存在，存在则修改文件名
	if _, err = os.Stat(opt.Output); err == nil {
		nameExt := opt.Output[len(opt.Output)-4:]
		randSuffix := time.Now().Unix()
		opt.Output = fmt.Sprintf("%s_%d%s", opt.Output[:len(opt.Output)-4], randSuffix, nameExt)
		fmt.Printf("output file exists, changed to %s\n", opt.Output)
	}
	return api.WriteContextFile(ctx, opt.Output)
}
