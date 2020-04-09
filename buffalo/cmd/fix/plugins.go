package fix

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gobuffalo/buffalo/genny/plugins/install"
	"github.com/gobuffalo/buffalo/internal/takeon/github.com/markbates/errx"
	"github.com/gobuffalo/buffalo/plugins/plugdeps"
	"github.com/gobuffalo/genny/v2"
	"github.com/gobuffalo/meta"
)

// plugins will fix plugins between releases
func plugins(r *Runner) error {
	fmt.Println("~~~ Cleaning plugins cache ~~~")
	os.RemoveAll(plugins.CachePath)
	plugs, err := plugdeps.List(r.App)
	if err != nil && (errx.Unwrap(err) != plugdeps.ErrMissingConfig) {
		return err
	}

	run := genny.WetRunner(context.Background())
	gg, err := install.New(&install.Options{
		App:     r.App,
		Plugins: plugs.List(),
	})

	run.WithGroup(gg)

	fmt.Println("~~~ Reinstalling plugins ~~~")
	return run.Run()
}

// removeOldPlugins will remove old Pop plugin
func removeOldPlugins(r *Runner) error {
	fmt.Println("~~~ Removing old plugins ~~~")

	run := genny.WetRunner(context.Background())
	app := meta.New(".")
	plugs, err := plugdeps.List(app)
	if err != nil && (errx.Unwrap(err) != plugdeps.ErrMissingConfig) {
		return err
	}

	a := strings.TrimSpace("github.com/gobuffalo/buffalo-pop")
	bin := path.Base(a)
	plugs.Remove(plugdeps.Plugin{
		Binary: bin,
		GoGet:  a,
	})

	fmt.Println("~~~ Removing github.com/gobuffalo/buffalo-pop plugin ~~~")

	run.WithRun(func(r *genny.Runner) error {
		p := plugdeps.ConfigPath(app)
		bb := &bytes.Buffer{}
		if err := plugs.Encode(bb); err != nil {
			return err
		}
		return r.File(genny.NewFile(p, bb))
	})
	if err != nil {
		return err
	}

	return run.Run()
}
