package integration

import (
//	"bytes"
//	"context"
//	"crypto/rand"
//	"crypto/sha256"
//	"fmt"
	"log"
//	"os"
//	"io"
//	"sync"
//	"testing"
//	"strings"
//	"slices"

//	"k8s.io/apimachinery/pkg/labels"
//	"k8s.io/apimachinery/pkg/api/resource"
//	"k8s.io/client-go/kubernetes"
//	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
//	"k8s.io/client-go/rest"
//	"k8s.io/client-go/tools/clientcmd"
//	"k8s.io/client-go/tools/remotecommand"
	ctrlrt "sigs.k8s.io/controller-runtime"
//	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlrtlog "sigs.k8s.io/controller-runtime/pkg/log"
	"github.com/go-logr/logr"

	// Options
//	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
//	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
//	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
//	coreapi "k8s.io/api/core/v1"
//	storapi "k8s.io/api/storage/v1"
//	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
//	apiruntime "k8s.io/apimachinery/pkg/runtime"
//	kubescheme "k8s.io/client-go/kubernetes/scheme"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
