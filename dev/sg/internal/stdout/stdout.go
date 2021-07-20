package stdout

import (
	"os"

	"github.com/sourcegraph/sourcegraph/lib/output"
)

var Out = output.NewOutput(os.Stdout, output.OutputOpts{
	ForceColor: true,
	ForceTTY:   true,
})
