package build

import (
	"strings"
	"testing"

	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/meta"
	"github.com/stretchr/testify/require"
)

func Test_buildCmd(t *testing.T) {
	envy.Set("GO_BIN", "go")
	r := require.New(t)

	opts := &Options{
		App: meta.New("."),
	}
	c, err := buildCmd(opts)
	r.NoError(err)
	r.Equal("go build -o bin/build", strings.Join(c.Args, " "))

	opts.Environment = "bar"
	c, err = buildCmd(opts)
	r.NoError(err)
	r.Equal("go build -tags bar -o bin/build", strings.Join(c.Args, " "))

	opts.App.Bin = "bin/foo"
	c, err = buildCmd(opts)
	r.NoError(err)
	r.Equal("go build -tags bar -o bin/foo", strings.Join(c.Args, " "))

	opts.WithSQLite = true
	c, err = buildCmd(opts)
	r.NoError(err)
	r.Equal("go build -tags bar sqlite -o bin/foo", strings.Join(c.Args, " "))

	opts.LDFlags = "-X foo.Bar=baz"
	c, err = buildCmd(opts)
	r.NoError(err)
	r.Equal("go build -tags bar sqlite -o bin/foo -ldflags -X foo.Bar=baz", strings.Join(c.Args, " "))

	opts.Static = true
	c, err = buildCmd(opts)
	r.NoError(err)
	r.Equal("go build -tags bar sqlite -o bin/foo -ldflags -linkmode external -extldflags \"-static\" -X foo.Bar=baz", strings.Join(c.Args, " "))

	opts.LDFlags = "-X main.BuildTime=asdf"
	_, err = buildCmd(opts)
	r.Error(err)

	opts.LDFlags = "-X main.BuildVersion=asdf"
	_, err = buildCmd(opts)
	r.Error(err)
}