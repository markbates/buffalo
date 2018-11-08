package build

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gobuffalo/buffalo/generators/assets/webpack"
	"github.com/gobuffalo/envy"
	"github.com/stretchr/testify/require"
)

func Test_assets(t *testing.T) {
	r := require.New(t)

	opts := &Options{
		WithAssets: true,
	}
	r.NoError(opts.Validate())
	opts.App.WithWebpack = true

	run := cokeRunner()
	run.WithNew(assets(opts))

	envy.MustSet("NODE_ENV", "")
	ne := envy.Get("NODE_ENV", "")
	r.Empty(ne)
	r.NoError(run.Run())

	ne = envy.Get("NODE_ENV", "")
	r.NotEmpty(ne)
	r.Equal(opts.Environment, ne)

	res := run.Results()

	cmds := []string{webpack.BinPath}
	r.Len(res.Commands, len(cmds))
	for i, c := range res.Commands {
		r.Equal(cmds[i], strings.Join(c.Args, " "))
	}

	s, err := res.Find("../public/assets/dummy")
	if err != nil || s.String() != "placeholder for static builds" {
		panic(fmt.Sprintf("%v", s.String()))
	}
}

func Test_assets_Archived(t *testing.T) {
	r := require.New(t)

	opts := &Options{
		WithAssets:    true,
		ExtractAssets: true,
	}
	r.NoError(opts.Validate())

	run := cokeRunner()
	opts.Root = run.Root
	run.WithNew(assets(opts))
	r.NoError(run.Run())

	res := run.Results()

	cmds := []string{}
	r.Len(res.Commands, len(cmds))
	for i, c := range res.Commands {
		r.Equal(cmds[i], strings.Join(c.Args, " "))
	}

	// r.Len(res.Files, 1)

	f, err := res.Find("actions/app.go")
	r.NoError(err)
	r.Contains(f.String(), `// app.ServeFiles("/"`)
}
