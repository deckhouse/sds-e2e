package integration

import (
	"log"

	"github.com/go-logr/logr"
	ctrlrt "sigs.k8s.io/controller-runtime"
	ctrlrtlog "sigs.k8s.io/controller-runtime/pkg/log"
)

/*  Logs  */

func initLog() {
	ctrlrt.SetLogger(logr.New(ctrlrtlog.NullLogSink{}))
}

func Debugf(format string, v ...any) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("    \033[32m🦗")
	log.Printf("\033[m"+format+"\033[0m", v...)
}

func Infof(format string, v ...any) {
	log.SetFlags(0)
	//log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("    \033[2m✎ ")
	log.Printf("\033[2m"+format+"\033[0m", v...)
}

func Warnf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix("    \033[93m🗈 ")
	log.Printf("\033[0;2m"+format+"\033[0m", v...)
}

func Errf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix("    \033[91m❕")
	log.Printf("\033[0m"+format+"\033[0m", v...)
}

func Critf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix("    \033[91;5m🔥")
	log.Printf("\033[0;91m"+format+"\033[0m", v...)
}

/*  Kuber Client  */

var clrCache = map[string]*KCluster{}

func GetCluster(configPath, clusterName string) *KCluster {
	k := configPath + ":" + clusterName
	if _, ok := clrCache[k]; !ok {
		clr, err := InitKCluster(configPath, clusterName)
		if err != nil {
			Critf("Kubeclient '%s' problem", k)
			panic(err)
			// or return nil
		}
		clrCache[k] = clr
	}
	return clrCache[k]
}
