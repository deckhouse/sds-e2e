package integration

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"time"
)

func hashMd5(in string) string {
	binHash := md5.Sum([]byte(in))
	return hex.EncodeToString(binHash[:])
}

func base64Encode(in string) string {
	return base64.StdEncoding.EncodeToString([]byte(in))
}

var lastLog = time.Now()

func RetrySec(sec int, f func() error) error {
	latency := time.Duration((sec + 15) / 7)
	if latency > 15 {
		latency = 15
	}

	start, lastMsg := time.Now(), ""
	for i := 0; ; i++ {
		err := f()
		if err == nil {
			return nil
		}
		if int(time.Since(start).Seconds()) > sec {
			Errf("Retry timeout %ds: %s", sec, err.Error())
			return err
		}
		if (time.Since(lastLog) > 10*time.Second && lastMsg != err.Error()) ||
			time.Since(lastLog) > 2*time.Minute {
			Debugf(err.Error())
			lastLog, lastMsg = time.Now(), err.Error()
		}
		time.Sleep(latency * time.Second)
	}
	return nil
}
