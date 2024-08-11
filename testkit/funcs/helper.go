package funcs

import (
	"fmt"
	"log"
)

type patchUInt32Value struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func LogFatalIfError(err error, out string, exclude ...string) {
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		if len(exclude) > 0 {
			for _, excludeError := range exclude {
				if err.Error() == excludeError {
					return
				}
			}
		}
		log.Fatal(err.Error())
	}
}
