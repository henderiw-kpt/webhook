package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/henderiw-kpt/webhook/transformer"
)

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(transformer.Run)); err != nil {
		os.Exit(1)
	}
}
