package engine

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type Options struct {
	Input    string
	Output   string
	Start    time.Time
	End      time.Time
	Fallback string
	Expired  string
	NoPrint  bool
	NoCopy   bool
}

func Run(opts Options) error {
	tmp := opts.Output + ".tmp"

	ctx, err := api.ReadContextFile(opts.Input)
	if err != nil {
		return err
	}

	// 1. OCG
	ocgs, err := registerOCGs(ctx)
	if err != nil {
		return err
	}
	injectOCProperties(ctx, ocgs)

	// 2. Pages
	for p := 1; p <= ctx.PageCount; p++ {
		pageDict, _, _, err := ctx.PageDict(p, false)
		if err != nil {
			return err
		}
		if pageDict == nil {
			continue
		}

		injectOCGResources(ctx, pageDict, ocgs)

		normal, err := readPageContent(ctx, pageDict)
		if err != nil {
			return err
		}
		fallback := textBlock(opts.Fallback)
		expired := textBlock(opts.Expired)

		if err := rewritePageContent(ctx, pageDict, normal, fallback, expired); err != nil {
			return err
		}
	}

	// 3. Form probe
	if err := injectTagField(ctx); err != nil {
		return err
	}

	// 4. JavaScript
	injectJS(ctx, opts.Start, opts.End)

	// 5. Permissions
	if opts.NoPrint || opts.NoCopy {
		restrictPermissions(ctx, opts.NoPrint, opts.NoCopy)
	}

	// 6. Atomic write
	// Normalize XRefTable entries: replace any *types.StreamDict pointers (possibly nested)
	// with value types.StreamDict to satisfy pdfcpu's write type switches.
	var normalize func(o types.Object) types.Object
	normalize = func(o types.Object) types.Object {
		switch v := o.(type) {
		case *types.StreamDict:
			return *v
		case types.Dict:
			for k, vv := range v {
				v[k] = normalize(vv)
			}
			return v
		case types.Array:
			for i, vv := range v {
				v[i] = normalize(vv)
			}
			return v
		default:
			return o
		}
	}

	// Normalize entire xref table by iterating all registered entries.
	fmt.Fprintf(os.Stderr, "xref table entries: %d\n", len(ctx.Table))
	for objNr, entry := range ctx.Table {
		if entry == nil || entry.Object == nil {
			continue
		}
		// Debug pre-normalize
		if objNr == 50 {
			fmt.Fprintf(os.Stderr, "pre-normalize obj#50 type=%T\n", entry.Object)
		}
		entry.Object = normalize(entry.Object)

		// If entry.Object is still a pointer type, attempt reflection-based deref.
		rv := reflect.ValueOf(entry.Object)
		if rv.Kind() == reflect.Ptr && rv.Elem().IsValid() {
			v := rv.Elem().Interface()
			if to, ok := v.(types.Object); ok {
				entry.Object = to
				fmt.Fprintf(os.Stderr, "reflect-converted obj#%d to %T\n", objNr, entry.Object)
			}
		}

		// Debug print occasionally to stderr for manual inspection.
		if objNr%10 == 0 {
			fmt.Fprintf(os.Stderr, "normalize: obj#%d type=%T\n", objNr, entry.Object)
		}
	}

	if e50, ok := ctx.FindTableEntryLight(50); ok && e50 != nil && e50.Object != nil {
		fmt.Fprintf(os.Stderr, "post-loop obj#50 type=%T\n", e50.Object)
		if _, ok := e50.Object.(*types.StreamDict); ok {
			fmt.Fprintf(os.Stderr, "post-loop obj#50 still pointer to StreamDict\n")
		}
	}

	if err := api.WriteContextFile(ctx, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, opts.Output)
}
