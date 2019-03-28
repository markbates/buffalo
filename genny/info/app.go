package info

import (
	"fmt"
	"reflect"

	"github.com/gobuffalo/genny"
)

func appDetails(opts *Options) genny.RunFn {
	return func(r *genny.Runner) error {
		opts.Out.Header("Buffalo: Application Details")
		rv := reflect.ValueOf(opts.App)
		rt := rv.Type()

		var lines [][]string
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !rv.FieldByName(f.Name).CanInterface() {
				continue
			}

			v := rv.FieldByName(f.Name).Interface()
			line := []string{f.Name, fmt.Sprint(v)}

			lines = append(lines, line)
		}
		return opts.Out.Tabs(lines)
	}
}
