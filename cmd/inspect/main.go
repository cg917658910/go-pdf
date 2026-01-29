package main

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func main() {
	fname := "m2_protected.pdf"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	ctx, err := api.ReadContextFile(fname)
	if err != nil {
		fmt.Printf("read context %s: %v\n", fname, err)
		os.Exit(1)
	}

	fmt.Printf("Pages: %d\n", ctx.PageCount)
	if ocp, found := ctx.RootDict.Find("OCProperties"); found {
		fmt.Printf("OCProperties: %T %v\n", ocp, ocp)
	} else {
		fmt.Printf("OCProperties: <nil>\n")
	}

	for p := 1; p <= ctx.PageCount; p++ {
		page, _, _, err := ctx.PageDict(p, true)
		if err != nil {
			fmt.Printf("page %d: error: %v\n", p, err)
			continue
		}
		fmt.Printf("\n--- Page %d ---\n", p)
		if c, ok := page.Find("Contents"); ok {
			// handle common cases including StreamDict
			switch c := c.(type) {
			case types.IndirectRef:
				fmt.Printf("Contents: IndirectRef %v\n", c)
			case types.Array:
				fmt.Printf("Contents: Array len=%d\n", len(c))
			case types.StreamDict:
				fmt.Printf("Contents: StreamDict (inline)\n")
			default:
				fmt.Printf("Contents: %T\n", c)
			}
		} else {
			fmt.Printf("Contents: <nil>\n")
		}

		if res, ok := page.Find("Resources"); ok {
			if rd, err := ctx.DereferenceDict(res); err == nil {
				if xo, found := rd.Find("XObject"); found {
					if xd, err := ctx.DereferenceDict(xo); err == nil {
						fmt.Printf("Resources.XObject keys:\n")
						for k := range xd {
							fmt.Printf(" - %s\n", k)
						}
					} else {
						fmt.Printf("Resources.XObject: %T (could not deref)\n", xo)
					}
				} else {
					fmt.Printf("Resources: no XObject\n")
				}
			} else {
				fmt.Printf("Resources: could not deref dict: %v\n", err)
			}
		} else {
			fmt.Printf("Resources: <nil>\n")
		}

		if ann, found := page.Find("Annots"); found {
			if arr, err := ctx.DereferenceArray(ann); err == nil {
				fmt.Printf("Annots: count=%d\n", len(arr))
				for i, a := range arr {
					fmt.Printf(" Annot %d: %T %v\n", i, a, a)
					if ar, ok := a.(types.IndirectRef); ok {
						if ad, err := ctx.DereferenceDict(ar); err == nil {
							if nm, found := ad.Find("NM"); found {
								fmt.Printf("  NM: %v\n", nm)
							}
							if ap, found := ad.Find("AP"); found {
								fmt.Printf("  AP present: %T %v\n", ap, ap)
								if apd, err := ctx.DereferenceDict(ap); err == nil {
									if n, ok := apd.Find("N"); ok {
										fmt.Printf("   AP.N: %T %v\n", n, n)
									}
								}
							}
							if oc, found := ad.Find("OC"); found {
								fmt.Printf("  Annot.OC: %T %v\n", oc, oc)
							}
						}
					}
				}
			} else {
				fmt.Printf("Annots: could not deref: %v\n", err)
			}
		} else {
			fmt.Printf("Annots: <nil>\n")
		}
	}

	// print AcroForm fields if any
	if af, found := ctx.RootDict.Find("AcroForm"); found {
		fmt.Printf("AcroForm: %T %v\n", af, af)
		if fd, err := ctx.DereferenceDict(af); err == nil {
			if fields, found := fd.Find("Fields"); found {
				if arr, err := ctx.DereferenceArray(fields); err == nil {
					fmt.Printf("AcroForm.Fields count=%d\n", len(arr))
					for i, f := range arr {
						fmt.Printf(" Field %d: %v\n", i, f)
					}
				}
			}
		}
	}
}
