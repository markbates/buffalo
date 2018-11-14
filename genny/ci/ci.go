package ci

import (
	"html/template"

	"github.com/gobuffalo/genny"
	"github.com/gobuffalo/genny/movinglater/gotools"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
)

// New generator for adding travis or gitlab
func New(opts *Options) (*genny.Generator, error) {
	g := genny.New()

	if err := opts.Validate(); err != nil {
		return g, errors.WithStack(err)
	}

	g.Transformer(genny.Replace("-no-pop", ""))
	g.Transformer(genny.Dot())

	box := packr.NewBox("../ci/templates")

	var fname string
	switch opts.Provider {
	case "travis", "travis-ci":
		fname = "-dot-travis.yml.tmpl"
	case "gitlab", "gitlab-ci":
		if opts.App.WithPop {
			fname = "-dot-gitlab-ci.yml.tmpl"
		} else {
			fname = "-dot-gitlab-ci-no-pop.yml.tmpl"
		}
	default:
		return g, errors.Errorf("could not find a template for %s", opts.Provider)
	}

	f, err := box.FindString(fname)
	if err != nil {
		return g, errors.WithStack(err)
	}

	g.File(genny.NewFileS(fname, f))

	data := map[string]interface{}{
		"opts": opts,
	}

	if opts.DBType == "postgres" {
		data["testDbUrl"] = "postgres://postgres:postgres@postgres:5432/" + opts.App.Name.File().String() + "_test?sslmode=disable"
	} else if opts.DBType == "mysql" {
		data["testDbUrl"] = "mysql://root:root@(mysql:3306)/" + opts.App.Name.File().String() + "_test?parseTime=true&multiStatements=true&readTimeout=1s"
	} else {
		data["testDbUrl"] = ""
	}

	helpers := template.FuncMap{}

	t := gotools.TemplateTransformer(data, helpers)
	g.Transformer(t)

	return g, nil
}