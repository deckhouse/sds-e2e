package test

import (
	"gocontainer/funcs"
	"testing"
)

func TestCreateStsLogs(t *testing.T) {
	funcs.CreateLogSts(*clientset)
}
