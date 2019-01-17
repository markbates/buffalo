// +build integration_test

package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/pop"
	"github.com/stretchr/testify/require"
)

func Test_NewCmd_NoName(t *testing.T) {
	err := call([]string{"new"}, nil)
	r := require.New(t)
	r.Error(err)
	r.EqualError(err, "you must enter a name for your new application")
}

func Test_NewCmd_InvalidDBType(t *testing.T) {
	args := []string{
		"new",
		"coke",
		"--db-type",
		"a",
	}
	err := call(args, nil)
	r := require.New(t)
	r.Error(err)
	r.EqualError(err, fmt.Sprintf(`unknown dialect "a" expecting one of %s`, strings.Join(pop.AvailableDialects, ", ")))
}

func Test_NewCmd_ForbiddenAppName(t *testing.T) {
	args := []string{
		"new",
		"buffalo",
	}
	err := call(args, nil)
	r := require.New(t)
	r.Error(err)
	r.EqualError(err, "name buffalo is not allowed, try a different application name")
}

func Test_NewCmd_Nominal(t *testing.T) {
	if envy.Get("GO111MODULE", "off") == "on" {
		t.Skip("CURRENTLY NOT SUPPORTED")
	}
	r := require.New(t)
	args := []string{
		"new",
		"hello_world",
		"--skip-pop",
		"--skip-webpack",
		"--vcs=none",
	}
	err := call(args, func(tdir string) {
		r.DirExists(filepath.Join(tdir, "hello_world"))
	})
	r.NoError(err)

}

func Test_NewCmd_API(t *testing.T) {
	if envy.Get("GO111MODULE", "off") == "on" {
		t.Skip("CURRENTLY NOT SUPPORTED")
	}
	args := []string{
		"new",
		"hello_world",
		"--skip-pop",
		"--api",
		"--vcs=none",
	}
	r := require.New(t)
	err := call(args, func(tdir string) {
		r.DirExists(filepath.Join(tdir, "hello_world"))
	})
	r.NoError(err)

}

// func Test_NewCmd_WithDep(t *testing.T) {
// 	if envy.Get("GO111MODULE", "off") == "on" {
// 		t.Skip("CURRENTLY NOT SUPPORTED")
// 	}
// 	c := RootCmd
//
// 	r := require.New(t)
// 	gp, err := envy.MustGet("GOPATH")
// 	r.NoError(err)
//
// 	newApp := func(rr *require.Assertions) {
// 		cpath := filepath.Join(gp, "src", "github.com", "gobuffalo")
// 		tdir, err := ioutil.TempDir(cpath, "testapp")
// 		rr.NoError(err)
// 		defer os.RemoveAll(tdir)
//
// 		pwd, err := os.Getwd()
// 		rr.NoError(err)
// 		os.Chdir(tdir)
// 		defer os.Chdir(pwd)
//
// 		c.SetArgs([]string{
// 			"new",
// 			"hello_world",
// 			"--skip-pop",
// 			"--skip-webpack",
// 			"--with-dep",
// 			"--vcs=none",
// 		})
// 		err = c.Execute()
// 		rr.NoError(err)
//
// 		rr.DirExists(filepath.Join(tdir, "hello_world"))
// 		rr.FileExists(filepath.Join(tdir, "hello_world", "Gopkg.toml"))
// 		rr.FileExists(filepath.Join(tdir, "hello_world", "Gopkg.lock"))
// 		rr.DirExists(filepath.Join(tdir, "hello_world", "vendor"))
// 	}
//
// 	t.Run("without dep in PATH", func(tt *testing.T) {
// 		if runtime.GOOS == "windows" {
// 			tt.Skip("Skipping on Windows")
// 		}
// 		rr := require.New(tt)
// 		if dep, err := exec.LookPath("dep"); err == nil {
// 			rr.NoError(os.Remove(dep))
// 		}
// 		newApp(rr)
// 	})
//
// 	t.Run("with dep in PATH", func(tt *testing.T) {
// 		rr := require.New(tt)
// 		newApp(rr)
// 	})
// }

func Test_NewCmd_WithPopSQLite3(t *testing.T) {
	if envy.Get("GO111MODULE", "off") == "on" {
		t.Skip("CURRENTLY NOT SUPPORTED")
	}
	r := require.New(t)

	args := []string{
		"new",
		"hello_world",
		"--db-type=sqlite3",
		"--skip-webpack",
		"--vcs=none",
	}

	err := call(args, func(tdir string) {
		r.DirExists(filepath.Join(tdir, "hello_world"))
		r.FileExists(filepath.Join(tdir, "hello_world", "database.yml"))
	})
	r.NoError(err)

}
