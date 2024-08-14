package cluster_create

import (
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/melbahja/goph"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestGetConfig(t *testing.T) {
	auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
	funcs.LogFatalIfError(err, "")

	var masterClient *goph.Client
	masterClient = funcs.GetSSHClient(funcs.MasterNodeIP, "user", auth)
	defer masterClient.Close()

	validTokenExists := false
	for !validTokenExists {
		out, err := masterClient.Run(fmt.Sprintf("sudo -i /bin/bash %s", filepath.Join(funcs.RemoteAppPath, funcs.UserCreateScriptName)))
		funcs.LogFatalIfError(err, string(out))
		out, err = masterClient.Run(fmt.Sprintf("cat %s", filepath.Join(funcs.RemoteAppPath, funcs.KubeConfigName)))
		funcs.LogFatalIfError(err, string(out))
		var validBase64 = regexp.MustCompile(`token: [A–Za-z0-9\+/=-_\.]{10,}`)
		validTokenExists = validBase64.MatchString(string(out))
		time.Sleep(10 * time.Second)
	}

	funcs.LogFatalIfError(masterClient.Download(filepath.Join(funcs.RemoteAppPath, funcs.KubeConfigName), filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName)), "")
}
