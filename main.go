// Command probe-db serves a Probe action that runs SQL against MySQL,
// PostgreSQL and SQLite. Probe starts it for a step that uses
// github.com/mozership/probe-db@<commit>.
package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/mozership/probe/sdk"
)

func main() {
	sdk.Serve(func(log hclog.Logger) sdk.Action {
		return &Action{log: log}
	})
}
