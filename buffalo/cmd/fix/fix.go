package fix

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var replace = map[string]string{
	"github.com/markbates/pop":      "github.com/gobuffalo/pop",
	"github.com/markbates/validate": "github.com/gobuffalo/validate",
	"github.com/satori/go.uuid":     "github.com/gobuffalo/uuid",

	"github.com/gobuffalo/buffalo/middleware/basicauth": "github.com/gobuffalo/mw-basicauth",
	"github.com/gobuffalo/buffalo/middleware/csrf":      "github.com/gobuffalo/mw-csrf",
	"github.com/gobuffalo/buffalo/middleware/i18n":      "github.com/gobuffalo/mw-i18n",
	"github.com/gobuffalo/buffalo/middleware/ssl":       "github.com/gobuffalo/mw-ssl",
	"github.com/gobuffalo/buffalo/middleware/tokenauth": "github.com/gobuffalo/mw-tokenauth",
}

var ic = ImportConverter{
	Data: replace,
	Aliases: map[string]string{
		"github.com/gobuffalo/mw-basicauth": "basicauth",
		"github.com/gobuffalo/mw-csrf":      "csrf",
		"github.com/gobuffalo/mw-i18n":      "i18n",
		"github.com/gobuffalo/mw-ssl":       "ssl",
		"github.com/gobuffalo/mw-tokenauth": "tokenauth",
	},
}

var checks = []Check{
	ic.Process,
	WebpackCheck,
	PackageJSONCheck,
	DepEnsure,
	DeprecrationsCheck,
}

func ask(q string) bool {
	fmt.Printf("? %s [y/n]\n", q)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	text = strings.ToLower(strings.TrimSpace(text))
	return text == "y" || text == "yes"
}
