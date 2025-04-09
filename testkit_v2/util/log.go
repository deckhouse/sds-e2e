package integration

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

func getPrefix() string {
	return "    "
}

func getDuration() string {
	if runtime.GOOS != "linux" {
		return fmt.Sprintf("%dm", int(time.Since(startTime).Minutes()))
	}

	char := int('🯰')
	i := int(time.Since(startTime).Seconds())
	if i >= 1000 {
		i = i / 60
		return fmt.Sprintf("%c%cm", rune(char+i%100/10), rune(char+i%10))
	}
	return fmt.Sprintf("%c%c%c", rune(char+i%1000/100), rune(char+i%100/10), rune(char+i%10))
}

func Filelogf(format string, v ...any) {
	if fileLogger == nil {
		return
	}
	fileLogger.Printf(format, v...)
}

func Debugf(format string, v ...any) {
	Filelogf("🦗"+format, v...)
	if !*debugFlag {
		return
	}
	log.SetFlags(0)
	//log.SetFlags(log.Lmicroseconds)
	log.SetPrefix(getPrefix())
	log.Printf("\033[32m🦗\033[2m"+getDuration()+" \033[0m"+format+"\033[0m", v...)
}

func Infof(format string, v ...any) {
	Filelogf("✎ "+format, v...)
	if !*verboseFlag && !*debugFlag {
		return
	}
	log.SetFlags(0)
	//log.SetFlags(log.Lmicroseconds)
	log.SetPrefix(getPrefix())
	log.Printf("\033[2m✎ "+getDuration()+" \033[2m"+format+"\033[0m", v...)
}

func Warnf(format string, v ...any) {
	Filelogf("🗈 "+format, v...)
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Printf("\033[93m🗈 \033[2m"+getDuration()+" \033[0;2m"+format+"\033[0m", v...)
}

func Warn(v ...any) {
	Warnf(fmt.Sprint(v...))
}

func Errorf(format string, v ...any) {
	Filelogf("❕"+format, v...)
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Printf("\033[91m❕\033[2m"+getDuration()+" \033[0m"+format+"\033[0m", v...)
}

func Critf(format string, v ...any) {
	Filelogf("⚠️ "+format, v...)
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Printf("\033[91;5m⚠️ \033[2m"+getDuration()+" \033[0;91m"+format+"\033[0m", v...)
}

func Fatalf(format string, v ...any) {
	Filelogf("🯀 "+format, v...)
	log.SetFlags(0)
	log.SetPrefix(getPrefix())
	log.Fatalf("\033[31m🯀 "+getDuration()+" \033[0m"+format, v...)
}
