package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	pop "github.com/gobuffalo/buffalo-pop/v2/genny/newapp"
	"github.com/gobuffalo/buffalo/genny/assets/standard"
	"github.com/gobuffalo/buffalo/genny/assets/webpack"
	"github.com/gobuffalo/buffalo/genny/ci"
	"github.com/gobuffalo/buffalo/genny/docker"
	"github.com/gobuffalo/buffalo/genny/newapp/api"
	"github.com/gobuffalo/buffalo/genny/newapp/core"
	"github.com/gobuffalo/buffalo/genny/newapp/web"
	"github.com/gobuffalo/buffalo/genny/refresh"
	"github.com/gobuffalo/buffalo/genny/vcs"
	"github.com/gobuffalo/buffalo/internal/takeon/github.com/markbates/errx"
	"github.com/gobuffalo/envy"
	fname "github.com/gobuffalo/flect/name"
	"github.com/gobuffalo/genny/v2"
	"github.com/gobuffalo/genny/v2/gogen"
	"github.com/gobuffalo/logger"
	"github.com/gobuffalo/meta"
	"github.com/gobuffalo/packr/v2/plog"
	"github.com/gobuffalo/plush/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type newAppOptions struct {
	Options *core.Options
	Module  string
	Force   bool
	Verbose bool
	DryRun  bool
}

func parseNewOptions(args []string) (newAppOptions, error) {
	nopts := newAppOptions{
		Force:   viper.GetBool("force"),
		Verbose: viper.GetBool("verbose"),
		DryRun:  viper.GetBool("dry-run"),
		Module:  viper.GetString("module"),
	}

	if len(args) == 0 {
		return nopts, fmt.Errorf("you must enter a name for your new application")
	}
	if configError != nil {
		return nopts, configError
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nopts, err
	}
	app := meta.New(pwd)
	app.WithGrifts = true
	app.Name = fname.New(args[0])
	app.Bin = filepath.Join("bin", app.Name.String())

	if app.Name.String() == "." {
		app.Name = fname.New(filepath.Base(app.Root))
	} else {
		app.Root = filepath.Join(app.Root, app.Name.File().String())
	}

	if len(nopts.Module) == 0 {
		aa := meta.New(app.Root)
		app.PackageRoot(aa.PackagePkg)
	} else {
		app.PackageRoot(nopts.Module)
	}

	app.AsAPI = viper.GetBool("api")
	app.VCS = viper.GetString("vcs")

	app.WithPop = !viper.GetBool("skip-pop")
	app.WithWebpack = !viper.GetBool("skip-webpack")
	app.WithYarn = !viper.GetBool("skip-yarn")
	app.WithNodeJs = app.WithWebpack
	app.AsWeb = !app.AsAPI

	if app.AsAPI {
		app.WithWebpack = false
		app.WithYarn = false
		app.WithNodeJs = false
	}

	opts := &core.Options{}

	x := viper.GetString("docker")
	if len(x) > 0 && x != "none" {
		opts.Docker = &docker.Options{
			Style: x,
		}
		app.WithDocker = true
	}

	x = viper.GetString("ci-provider")
	if len(x) > 0 && x != "none" {
		opts.CI = &ci.Options{
			Provider: x,
			DBType:   viper.GetString("db-type"),
		}
	}

	if len(app.VCS) > 0 && app.VCS != "none" {
		opts.VCS = &vcs.Options{
			Provider: app.VCS,
		}
	}

	if app.WithPop {
		d := viper.GetString("db-type")
		if d == "sqlite3" {
			app.WithSQLite = true
		}

		opts.Pop = &pop.Options{
			Prefix:  app.Name.File().String(),
			Dialect: d,
		}
	}

	opts.Refresh = &refresh.Options{}

	opts.App = app
	nopts.Options = opts
	return nopts, nil
}

var configError error

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Creates a new Buffalo application",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Restore default values after usage (useful for testing)
		defer func() {
			cmd.Flags().Visit(func(f *pflag.Flag) {
				f.Value.Set(f.DefValue)
			})
			viper.BindPFlags(cmd.Flags())
		}()

		nopts, err := parseNewOptions(args)
		if err != nil {
			return err
		}

		opts := nopts.Options
		app := opts.App

		ctx := context.Background()

		run := genny.WetRunner(ctx)
		lg := logger.New(logger.DebugLevel)
		run.Logger = lg
		if nopts.Verbose {
			plog.Logger = lg
		}

		if nopts.DryRun {
			run = genny.DryRunner(ctx)
		}
		run.Root = app.Root
		if nopts.Force {
			os.RemoveAll(app.Root)
		}

		var gg *genny.Group

		if app.AsAPI {
			gg, err = api.New(&api.Options{
				Options: opts,
			})
		} else {
			wo := &web.Options{
				Options: opts,
			}
			if app.WithWebpack {
				wo.Webpack = &webpack.Options{}
			} else if !app.AsAPI {
				wo.Standard = &standard.Options{}
			}
			gg, err = web.New(wo)
		}
		if err != nil {
			if errx.Unwrap(err) == core.ErrNotInGoPath {
				return notInGoPath(app)
			}
			return err
		}
		run.WithGroup(gg)

		g := genny.New()
		g.Command(exec.Command("go", "mod", "tidy"))
		if err := run.With(g); err != nil {
			return nil
		}

		if err := run.WithNew(gogen.Fmt(app.Root)); err != nil {
			return err
		}

		// setup VCS last
		if opts.VCS != nil {
			// add the VCS generator
			if err := run.WithNew(vcs.New(opts.VCS)); err != nil {
				return err
			}
		}

		if err := run.Run(); err != nil {
			return err
		}

		run.Logger.Infof("Congratulations! Your application, %s, has been successfully built!", app.Name)
		run.Logger.Infof("You can find your new application at: %v", app.Root)
		run.Logger.Info("Please read the README.md file in your new application for next steps on running your application.")
		return nil
	},
}

func currentUser() (string, error) {
	if _, err := exec.LookPath("git"); err == nil {
		if b, err := exec.Command("git", "config", "github.user").Output(); err == nil {
			return string(b), nil
		}
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	username := u.Username
	if t := strings.Split(username, `\`); len(t) > 0 {
		username = t[len(t)-1]
	}
	return username, nil
}

func notInGoPath(app meta.App) error {
	username, err := currentUser()
	if err != nil {
		return err
	}
	pwd, _ := os.Getwd()
	t, err := plush.Render(notInGoWorkspace, plush.NewContextWith(map[string]interface{}{
		"name":     app.Name,
		"gopath":   envy.GoPath(),
		"current":  pwd,
		"username": username,
	}))
	if err != nil {
		return err
	}
	logrus.Error(t)
	os.Exit(-1)
	return nil
}

func init() {
	decorate("new", newCmd)
	RootCmd.AddCommand(newCmd)
	newCmd.Flags().Bool("api", false, "skip all front-end code and configure for an API server")
	newCmd.Flags().BoolP("force", "f", false, "delete and remake if the app already exists")
	newCmd.Flags().BoolP("dry-run", "d", false, "dry run")
	newCmd.Flags().BoolP("verbose", "v", false, "verbosely print out the go get commands")
	newCmd.Flags().Bool("skip-pop", false, "skips adding pop/soda to your app")
	newCmd.Flags().Bool("skip-webpack", false, "skips adding Webpack to your app")
	newCmd.Flags().Bool("skip-yarn", false, "use npm instead of yarn for frontend dependencies management")
	newCmd.Flags().String("db-type", "postgres", fmt.Sprintf("specify the type of database you want to use [%s]", strings.Join(pop.AvailableDialects, ", ")))
	newCmd.Flags().String("docker", "multi", "specify the type of Docker file to generate [none, multi, standard]")
	newCmd.Flags().String("ci-provider", "none", "specify the type of ci file you would like buffalo to generate [none, travis, gitlab-ci, circleci]")
	newCmd.Flags().String("vcs", "git", "specify the Version control system you would like to use [none, git, bzr]")
	newCmd.Flags().String("module", "", "specify the root module (package) name. [defaults to 'automatic']")
	viper.BindPFlags(newCmd.Flags())
	cfgFile := newCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.buffalo.yaml)")
	skipConfig := newCmd.Flags().Bool("skip-config", false, "skips using the config file")
	cobra.OnInitialize(initConfig(skipConfig, cfgFile))
}

func initConfig(skipConfig *bool, cfgFile *string) func() {
	return func() {
		if *skipConfig {
			return
		}

		var err error
		if *cfgFile != "" { // enable ability to specify config file via flag
			viper.SetConfigFile(*cfgFile)
			// Will error only if the --config flag is used
			if err = viper.ReadInConfig(); err != nil {
				configError = err
			}
		} else {
			viper.SetConfigName(".buffalo") // name of config file (without extension)
			viper.AddConfigPath("$HOME")    // adding home directory as first search path
			viper.AutomaticEnv()            // read in environment variables that match
			viper.ReadInConfig()
		}

	}
}

const notInGoWorkspace = `Oops! It would appear that you are not in your Go Workspace.

Your $GOPATH is set to "<%= gopath %>".

You are currently in "<%= current %>".

The standard location for putting Go projects is something along the lines of "$GOPATH/src/github.com/<%= username %>/<%= name %>" (adjust accordingly).

We recommend you go to "$GOPATH/src/github.com/<%= username %>/" and try "buffalo new <%= name %>" again.`
