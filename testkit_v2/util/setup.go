package integration

import (
	"log"
	"time"
)

//var (
//	startTime = time.Now()
//)

/*  Logs  */

func getPrefix() string {
	return "    "
}

func getDuration() string {
	i := int(time.Since(startTime).Seconds())
	return string('🯰' + i%1000 / 100) + string('🯰' + i%100 / 10) + string('🯰' + i%10)
	return "[" + time.Since(startTime).Round(time.Second).String() + "]"
}

func Debugf(format string, v ...any) {
	if !*debugFlag {
		return
	}
	//log.SetFlags(log.Lmicroseconds)
	log.SetPrefix(getPrefix())
	log.Printf("\033[32m🦗\033[2m"+getDuration()+" \033[0m"+format+"\033[0m", v...)
}

func Infof(format string, v ...any) {
	if !*verboseFlag && !*debugFlag {
		return
	}
	log.SetFlags(0)
	//log.SetFlags(log.Lmicroseconds)
	log.SetPrefix(getPrefix())
	log.Printf("\033[2m✎ "+getDuration()+" \033[2m"+format+"\033[0m", v...)
}

func Warnf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Printf("\033[93m🗈 \033[2m"+getDuration()+" \033[0;2m"+format+"\033[0m", v...)
}

func Errf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Printf("\033[91m❕\033[2m"+getDuration()+" \033[0m"+format+"\033[0m", v...)
}

func Critf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Printf("\033[91;5m🔥\033[2m"+getDuration()+" \033[0;91m"+format+"\033[0m", v...)
}

func Fatalf(format string, v ...any) {
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Fatalf("\033[31m"+getDuration()+" \033[0m"+format, v...)
}

/*  Kuber Client  */

var clrCache = map[string]*KCluster{}

func GetCluster(configPath, clusterName string) *KCluster {
	if len(clrCache) == 0 {
		envInit()
		if *standFlag == "metal" {
			ClusterCreate()
		}
	}

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
