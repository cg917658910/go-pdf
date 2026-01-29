package cli

import (
    "errors"
    "flag"
    "fmt"
    "time"

    "pdfguard/internal/engine"
)

func Execute() int {
    return protect()
}

func protect() int {
    var (
        in       = flag.String("in", "", "input pdf file")
        out      = flag.String("out", "", "output pdf file")
        startStr = flag.String("start", "", "start time RFC3339")
        endStr   = flag.String("end", "", "end time RFC3339")
        fallback = flag.String("fallback", "Unsupported viewer", "fallback text")
        expired  = flag.String("expired", "Expired", "expired text")
        noPrint  = flag.Bool("no-print", false, "disable printing")
        noCopy   = flag.Bool("no-copy", false, "disable copy")
    )
    flag.Parse()

    if err := validate(*in, *out, *startStr, *endStr); err != nil {
        fmt.Println("error:", err)
        return 1
    }

    start, _ := time.Parse(time.RFC3339, *startStr)
    end, _ := time.Parse(time.RFC3339, *endStr)

    opts := engine.Options{
        Input:    *in,
        Output:   *out,
        Start:    start,
        End:      end,
        Fallback: *fallback,
        Expired:  *expired,
        NoPrint:  *noPrint,
        NoCopy:   *noCopy,
    }

    if err := engine.Run(opts); err != nil {
        fmt.Println("error:", err)
        return 1
    }
    return 0
}

func validate(in, out, s, e string) error {
    if in == "" || out == "" {
        return errors.New("in/out required")
    }
    if _, err := time.Parse(time.RFC3339, s); err != nil {
        return errors.New("invalid start time")
    }
    if _, err := time.Parse(time.RFC3339, e); err != nil {
        return errors.New("invalid end time")
    }
    return nil
}
