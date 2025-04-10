/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
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
			Warnf("Retry %ds: %s", sec, err.Error())
			return err
		}
		if (time.Since(lastLog) > 10*time.Second && lastMsg != err.Error()) ||
			time.Since(lastLog) > 2*time.Minute {
			Debugf("Waiting... %s", err.Error())
			lastLog, lastMsg = time.Now(), err.Error()
		}
		time.Sleep(latency * time.Second)
	}
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
