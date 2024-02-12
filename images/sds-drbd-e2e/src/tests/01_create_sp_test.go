package test

import (
	"gocontainer/funcs"
	"testing"
)

func TestCreatePool(t *testing.T) {
	funcs.CreatePools(*dynamicClient)
}
