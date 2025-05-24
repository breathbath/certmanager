package main

import (
	"github.com/breathbath/certmanager/pkg/cmd"
	"github.com/breathbath/certmanager/pkg/errs"
	"github.com/breathbath/certmanager/pkg/logging"
)

func main() {
	logging.Init()

	err := cmd.Execute()
	errs.Handle(err, true)
}
