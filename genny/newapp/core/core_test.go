package core

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gobuffalo/buffalo-docker/genny/docker"
	"github.com/gobuffalo/buffalo/runtime"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/genny/gentest"
	"github.com/gobuffalo/meta"
	"github.com/stretchr/testify/require"
)

func init() {
	// normalize command output
	envy.Set("GO_BIN", "go")
}

func Test_New(t *testing.T) {
	r := require.New(t)

	app := meta.Named("coke", filepath.Join(envy.GoPath(), "src"))
	app.WithModules = false

	gg, err := New(&Options{
		App: app,
	})
	r.NoError(err)

	run := gentest.NewRunner()
	run.WithGroup(gg)

	r.NoError(run.Run())

	res := run.Results()

	cmds := []string{"go get github.com/gobuffalo/buffalo-plugins",
		"go get -t ./...",
	}
	r.Len(res.Commands, len(cmds))

	for i, c := range res.Commands {
		r.Equal(cmds[i], strings.Join(c.Args, " "))
	}

	expected := commonExpected
	for _, e := range expected {
		_, err = res.Find(e)
		r.NoError(err)
	}

	f, err := res.Find("actions/render.go")
	r.NoError(err)

	body := f.String()
	r.Contains(body, `r = render.New(render.Options{})`)
	unexpected := []string{
		"Dockerfile",
		"database.yml",
		"models/models.go",
		"go.mod",
		".buffalo.dev.yml",
		"assets/css/application.scss.css",
		"public/assets/application.js",
	}

	for _, u := range unexpected {
		_, err = res.Find(u)
		r.Error(err)
	}
}

func Test_New_Mods(t *testing.T) {
	r := require.New(t)

	app := meta.Named("coke", ".")
	(&app).PackageRoot("coke")
	app.WithModules = true

	gg, err := New(&Options{
		App: app,
	})
	r.NoError(err)

	run := gentest.NewRunner()
	run.WithGroup(gg)

	r.NoError(run.Run())

	res := run.Results()

	cmds := []string{
		"go mod init coke",
		"go get github.com/gobuffalo/buffalo-plugins",
		"go get github.com/gobuffalo/buffalo@" + runtime.Version,
		"go get",
		"go mod tidy",
	}
	r.Len(res.Commands, len(cmds))

	for i, c := range res.Commands {
		r.Equal(cmds[i], strings.Join(c.Args, " "))
	}

	expected := commonExpected
	for _, e := range expected {
		_, err = res.Find(e)
		r.NoError(err)
	}

	unexpected := []string{
		"Dockerfile",
		"database.yml",
		"models/models.go",
		".buffalo.dev.yml",
		"assets/css/application.scss.css",
		"public/assets/application.js",
	}
	for _, u := range unexpected {
		_, err = res.Find(u)
		r.Error(err)
	}

}

func Test_New_Docker(t *testing.T) {
	r := require.New(t)

	app := meta.New(".")
	app.WithModules = false

	gg, err := New(&Options{
		Docker: &docker.Options{},
	})
	r.NoError(err)

	run := gentest.NewRunner()
	run.WithGroup(gg)

	r.NoError(run.Run())

	res := run.Results()

	expected := append(commonExpected, "Dockerfile")
	for _, e := range expected {
		_, err := res.Find(e)
		r.NoError(err)
	}
}

var commonExpected = []string{
	"main.go",
	"actions/app.go",
	"actions/actions_test.go",
	"actions/render.go",
	"actions/home.go",
	"actions/home_test.go",
	"fixtures/sample.toml",
	"grifts/init.go",
	".codeclimate.yml",
	".env",
	"inflections.json",
	"README.md",
}
