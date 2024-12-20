package integration

import (
	"bytes"
	"context"
//	"crypto/rand"
//	"crypto/sha256"
	"fmt"
	"log"
//	"os"
//	"io"
//	"sync"
//	"testing"
	"strings"
	"slices"

//	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
//	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	ctrlrt "sigs.k8s.io/controller-runtime"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlrtlog "sigs.k8s.io/controller-runtime/pkg/log"
	"github.com/go-logr/logr"

	// Options
    snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
    srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
    virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	coreapi "k8s.io/api/core/v1"
    storapi "k8s.io/api/storage/v1"
    extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
    apiruntime "k8s.io/apimachinery/pkg/runtime"
    kubescheme "k8s.io/client-go/kubernetes/scheme"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//var (
////	jobConfigMux      sync.Mutex
////	prowComponentsMux sync.Mutex
//)

//TODO deprecated
//func NewClients(configPath, clusterName string) (ctrlrtclient.Client, error) {
//	cfg, err := NewRestConfig(configPath, clusterName)
//	if err != nil {
//		return nil, err
//	}
//	return ctrlrtclient.New(cfg, ctrlrtclient.Options{})
//}


func initLog() {
	ctrlrt.SetLogger(logr.New(ctrlrtlog.NullLogSink{}))
}

func NewRestConfig(configPath, clusterName string) (*rest.Config, error) {
	var cfg *rest.Config

	// use the current context in kubeconfig
	//	config, err = clientcmd.BuildConfigFromFlags("", configPath)
	//	if err != nil {
	//		return nil, err
	//	}

	var loader clientcmd.ClientConfigLoader
	if configPath != "" {
		loader = &clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}
	} else {
		loader = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
		overrides.CurrentContext = clusterName
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed create rest config: %w", err)
	}

	return cfg, nil
}

func NewKubeRTClient(configPath, clusterName string) (*ctrlrtclient.Client, error) {
	cfg, err := NewRestConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	// Add options
	var resourcesSchemeFuncs = []func(*apiruntime.Scheme) error {
		virt.AddToScheme,
		srv.AddToScheme,
		snc.AddToScheme,
		kubescheme.AddToScheme,
		extapi.AddToScheme,
		coreapi.AddToScheme,
		storapi.AddToScheme,
	}

    scheme := apiruntime.NewScheme()
    for _, f := range resourcesSchemeFuncs {
        err = f(scheme)
        if err != nil {
            return nil, err
        }
    }
    clientOpts := ctrlrtclient.Options{
        Scheme: scheme,
    }

	// Init client
	//clientOpts := ctrlrtclient.Options{}
	cl, err := ctrlrtclient.New(cfg, clientOpts)
	if err != nil {
		return nil, err
	}

	return &cl, nil
}

func NewKubeGoClient(configPath, clusterName string) (*kubernetes.Clientset, error) {
	// to create the clientset
	cfg, err := NewRestConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	cl, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

type KCluster struct {
	name string
	ctx context.Context
	rtClient *ctrlrtclient.Client
	goClient *kubernetes.Clientset
}

func InitKCluster(configPath, clusterName string) (*KCluster, error) {
	configPath = envConfigPath(configPath)
	clusterName	= envClusterName(clusterName)

	log.Println(fmt.Sprintf("Init Client %s/%s", configPath, clusterName))
	rcl, err := NewKubeRTClient(configPath, clusterName)
    if err != nil {
		log.Println(fmt.Sprintf("Can`t connect cluster %s", clusterName))
    	return nil, err
    }

	gcl, err := NewKubeGoClient(configPath, clusterName)
    if err != nil {
		log.Println(fmt.Sprintf("Can`t connect cluster %s", clusterName))
    	return nil, err
    }

	ctr := KCluster{
		name: clusterName,
		ctx: context.Background(),
		rtClient: rcl,
		goClient: gcl,
	}
	return &ctr, nil
}

func (clr *KCluster) GetNodes(filter map[string][]string) (resp []coreapi.Node, err error) {
	nodes, err := (*clr.goClient).CoreV1().Nodes().List(clr.ctx, metav1.ListOptions{})
    if err != nil {
		log.Println(fmt.Sprintf("Can`t get Nodes (%s)", clr.name))
        return nil, err
    }

	if filter == nil {
		return nodes.Items, nil
	}

	// Data filtering
	for _, node := range nodes.Items {
		img := node.Status.NodeInfo.OSImage
		valid := true

		if imgs, ok := filter["OS"]; ok {
			valid = false
			for _, i := range imgs {
				if strings.Contains(img, i) {
					valid = true
					break
				}
			}
		} else if imgs, ok := filter["notOS"]; ok {
			for _, i := range imgs {
				if strings.Contains(img, i) {
					valid = false
					break
				}
			}
		}
		if valid {
			resp = append(resp, node)
		}
	}

	return
}

func (clr *KCluster) GetBDs(filter map[string][]string) (resp []snc.BlockDevice, err error)  {
    bdList := &snc.BlockDeviceList{}
    err = (*clr.rtClient).List(clr.ctx, bdList)
    if err != nil {
		log.Println(fmt.Sprintf("Can`t get BDs (%s)", clr.name))
        return nil, err
    }

	if filter == nil {
		return bdList.Items, nil
	}
	for _, bd := range bdList.Items {
		if names, ok := filter["Name"]; ok {
			if slices.Contains(names, bd.Name) {
				resp = append(resp, bd)
				continue
			}
		} else if names, ok := filter["notName"]; ok {
			if slices.Contains(names, bd.Name) {
				continue
			}
			resp = append(resp, bd)
		}
	}

	return
}

func (clr *KCluster) GetLVG() ([]snc.LVMVolumeGroup, error) {
	lvgList := &snc.LVMVolumeGroupList{}
	err := (*clr.rtClient).List(clr.ctx, lvgList)
	if err != nil {
		log.Println(fmt.Sprintf("Can`t get LVGs (%s)", clr.name))
        return nil, err
	}

	return lvgList.Items, nil
}

func (clr *KCluster) GetTestLVG() (resp []snc.LVMVolumeGroup, err error) {
	lvgList, err := clr.GetLVG()
	if err != nil {
		return nil, err
	}

	for _, lvg := range lvgList {
		if lvg.Name[:8] == "e2e-lvg-" {
			resp = append(resp, lvg)
		}
	}
	return
}

func (clr *KCluster) AddLVG(nodeName, bdName string) error {
	lvgName := "e2e-lvg-" + nodeName[len(nodeName)-1:] + "-" + bdName[len(bdName)-3:]
	lvmVolumeGroup := &snc.LVMVolumeGroup{
	    ObjectMeta: metav1.ObjectMeta{
	        Name: lvgName,
	    },
	    Spec: snc.LVMVolumeGroupSpec{
	        ActualVGNameOnTheNode: lvgName,
	        BlockDeviceSelector: &metav1.LabelSelector{
	            MatchLabels: map[string]string{"kubernetes.io/metadata.name": bdName},
	        },
	        Type: "Local",
	        Local: snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
	    },
	}
	err := (*clr.rtClient).Create(clr.ctx, lvmVolumeGroup)
	if err != nil {
		log.Println(fmt.Sprintf("Can`t create LVG %s (node %s, bd %s)", lvgName, nodeName, bdName))
		return err
	}
	return nil
}

func (clr *KCluster) DelTestLVG() error {
	lvgList, _ := clr.GetTestLVG()

	for _, lvg := range lvgList {
		if err := (*clr.rtClient).Delete(clr.ctx, &lvg); err != nil {
			return err
		}
	}

	return nil
}

func (clr *KCluster) GetPVC(nsName string) ([]coreapi.PersistentVolumeClaim, error) {
    pvcList, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).List(clr.ctx, metav1.ListOptions{})
    //pvc, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), pvcName, metav1.GetOptions{})

//    pvcList := coreapi.PersistentVolumeClaimList{}
//    opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})
//    err := (*clr.rtClient).List(clr.ctx, &pvcList, opts)
    if err != nil {
		log.Println(fmt.Sprintf("Can`t get PVCs %s", nsName))
		return nil, err
    }

	return pvcList.Items, nil
}

func (clr *KCluster) GetTestPVC() ([]coreapi.PersistentVolumeClaim, error) {
    return clr.GetPVC("d8-upmeter")
    return clr.GetPVC("sds-replicated-volume-e2e-test")
}

func (clr *KCluster) UpdatePVC(pvc *coreapi.PersistentVolumeClaim) (*coreapi.PersistentVolumeClaim, error) {
	nsName := pvc.Namespace
//	return pvc, nil
	pvc, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).Update(clr.ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		log.Println(fmt.Sprintf("Can`t update PVC %s", pvc.Name))
		return pvc, err
    }

	return pvc, nil
}

func (clr *KCluster) GetVMD(nsName, vmdName string) ([]virt.VirtualDisk, error) {
	vmds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})

	err := (*clr.rtClient).List(clr.ctx, &vmds, opts)
	if err != nil {
		log.Println(fmt.Sprintf("Can`t get VMDs (%s, %s)", nsName, vmdName))
        return nil, err
	}

	if vmdName == "" {
		return vmds.Items, nil
	}

	vmdList := []virt.VirtualDisk{}
	for _, vmd := range vmds.Items {
		if vmdName == vmd.Name {  // <VM>-data|<VM>-system
			vmdList = append(vmdList, vmd)
		}
	}

	return vmdList, nil
}

func (clr *KCluster) GetTestVMD() ([]virt.VirtualDisk, error) {
	return clr.GetVMD(testNS, "")
}

func (clr *KCluster) UpdVMD(vmd *virt.VirtualDisk) error {
	return (*clr.rtClient).Update(clr.ctx, vmd)
}

// test NODE
/*
	fmt.Printf("node: %#v\n", nodes.Items[0].Name)
	node := nodes.Items[0]
	stdout, stderr, err := execNodeCmd(cfg2, cl2, &node, []string{"/bin/sh", "-c", "pwd"})
*/
// ----

func execNodeCmd(restCfg *rest.Config, clientset *kubernetes.Clientset, node *coreapi.Node, command []string) (string, string, error) {
//# Get standard bash shell
//kubectl node-shell <node>
    buf := &bytes.Buffer{}
    errBuf := &bytes.Buffer{}
    request := clientset.CoreV1().RESTClient().
        Post().
        Resource("nodes").
        //Name(node.Name).
        Name("d-shipkov-worker-0").
        SubResource("exec").
        VersionedParams(&coreapi.PodExecOptions{
            Command: command,
            Stdin:   false,
            Stdout:  true,
            Stderr:  true,
            TTY:     true,
        }, kubescheme.ParameterCodec)
	fmt.Printf("request.URL: %s\n", request.URL())
    exec, err := remotecommand.NewSPDYExecutor(restCfg, "POST", request.URL())
    if err != nil {
        return "", "", err
    }

    err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
        Stdout: buf,
        Stderr: errBuf,
    })
    if err != nil {
        return "", "", err
        return "", "", fmt.Errorf("%w Failed executing command %s on %v", err, command, node.Name)
    }

    // Return stdout, stderr.
    return buf.String(), errBuf.String(), nil
}

func execPodCmd(restCfg *rest.Config, clientset *kubernetes.Clientset, pod *coreapi.Pod, command []string) (string, string, error) {
    buf := &bytes.Buffer{}
    errBuf := &bytes.Buffer{}
    request := clientset.CoreV1().RESTClient().
        Post().
        Namespace(pod.Namespace).
        Resource("pods").
        Name(pod.Name).
        SubResource("exec").
        VersionedParams(&coreapi.PodExecOptions{
			//Container: "container",
            Command: command,
            Stdin:   false,
            Stdout:  true,
            Stderr:  true,
            TTY:     true,
        }, kubescheme.ParameterCodec)
    exec, err := remotecommand.NewSPDYExecutor(restCfg, "POST", request.URL())
    if err != nil {
        return "", "", err
    }

    err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
        Stdout: buf,
        Stderr: errBuf,
    })
    if err != nil {
        return "", "", fmt.Errorf("%w Failed executing command %s on %v/%v", err, command, pod.Namespace, pod.Name)
    }

    // Return stdout, stderr.
    return buf.String(), errBuf.String(), nil
}


// test POD
/*
	pods := &coreapi.PodList{}
	err = cl.List(ctx, pods, ctrlrtclient.InNamespace("test1"))
	if err != nil {
		t.Error("error listing pods", err)
	}
	fmt.Printf("pods: %#v\n", pods)

	cfg1, _ := tools.NewRestConfig("", "dev")
	cl1, _ := kubernetes.NewForConfig(cfg1)
	for _, ns := range []string{
//		"d8-sds-replicated-volume", "d8-system",
		"d8-ingress-nginx",
		//"d8-admission-policy-engine", "d8-cert-manager", "d8-chrony", "d8-cloud-instance-manager",
		//"d8-cloud-provider-yandex", "d8-cni-simple-bridge", "d8-dashboard", "d8-descheduler", "d8-ingress-nginx",
		//"d8-local-path-provisioner", "d8-log-shipper", "d8-monitoring", "d8-multitenancy-manager",
		//"d8-operator-prometheus", "d8-pod-reloader", "d8-sds-node-configurator", "d8-sds-replicated-volume", 
		//"d8-service-accounts", "d8-snapshot-controller", "d8-system", "d8-upmeter", "d8-user-authn", 
		//"d8-virtualization", "default", "kube-node-lease", "kube-public", "kube-system", "test1", 
	} {
		pods, err := cl1.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
    	if err != nil {
    	    t.Fatalf("could not list pods: %v", err)
    	}
		run_pods := map[string]bool{
//			"linstor-affinity-controller-68c97f94c9-mt6gt": true,
//			"linstor-scheduler-extender-764fb95bdf-g9f6p": true,
//			"spaas-f74f5f9c4-dr7p2": true,
//			"webhook-handler-6f6f654475-8l62t": true,

			"controller-nginx-hfn2t": true,
		}
		for _, pod := range pods.Items {
			if _, ok := run_pods[pod.Name]; !ok {continue}
			//v, err := pod.Marshal()
			//v, err := pod.Spec.Marshal()
			//fmt.Printf("P O D: %s\n%#v\n------------------------\n", string(v), err)

			//stdout, stderr, err := execPodCmd(cfg1, cl1, &pod, []string{"/bin/sh", "-c", "pwd"})
			//stdout, stderr, err := execPodCmd(cfg1, cl1, &pod, []string{"/usr/bin/bash", "-c", "pwd"})
			//stdout, stderr, err := execPodCmd(cfg1, cl1, &pod, []string{"/bin/sh", "-c", "sudo vgdisplay -C"})
			//if err != nil {
			//	//fmt.Printf("pod %s: ERROR\n", pod.Name)
			//} else {
			//	fmt.Printf("pod %s: %s, %s\n", pod.Name, string(stdout), string(stderr))
			//}
			//fmt.Printf("pod %s: %s, %s, %#v\n", pod.Name, string(stdout), string(stderr), err)
		}

		// ssh -i ~/.ssh/dshipkov ubuntu@d-shipkov-worker-2.ru-central1.internal sudo vgdisplay -C
	}

	// access the API to list pods
		//pods, _ := cl.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
		//fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	//fgsPods, err := clientset.CoreV1().Pods("default").List(context.TODO(),
    //    metav1.ListOptions{LabelSelector: "app=fakegitserver"})
*/
// ----




/*
func getPodLogs(clientset *kubernetes.Clientset, namespace, podName string, opts *coreapi.PodLogOptions) (string, error) {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("error in opening stream")
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf")
	}
	str := buf.String()

	return str, nil
}

func refreshProwPods(client ctrlrtclient.Client, ctx context.Context, name string) error {
	prowComponentsMux.Lock()
	defer prowComponentsMux.Unlock()

	var pods coreapi.PodList
	labels, _ := labels.Parse("app = " + name)
	if err := client.List(ctx, &pods, &ctrlrtclient.ListOptions{LabelSelector: labels}); err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if err := client.Delete(ctx, &pod); err != nil {
			return err
		}
	}
	return nil
}

// RandomString generates random string of 32 characters in length, and fail if it failed
func RandomString(t *testing.T) string {
	b := make([]byte, 512)
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("failed to generate random: %v", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b[:]))[:32]
}

func updateJobConfig(ctx context.Context, kubeClient ctrlrtclient.Client, filename string, rawConfig []byte) error {
	jobConfigMux.Lock()
	defer jobConfigMux.Unlock()

	var existingMap coreapi.ConfigMap
	if err := kubeClient.Get(ctx, ctrlrtclient.ObjectKey{
		Namespace: defaultNamespace,
		Name:      "job-config",
	}, &existingMap); err != nil {
		return err
	}

	if existingMap.BinaryData == nil {
		existingMap.BinaryData = make(map[string][]byte)
	}
	existingMap.BinaryData[filename] = rawConfig
	return kubeClient.Update(ctx, &existingMap)
}

// execRemoteCommand is the Golang-equivalent of "kubectl exec". The command
// string should be something like {"/bin/sh", "-c", "..."} if you want to run a
// shell script.
//
// Adapted from https://discuss.kubernetes.io/t/go-client-exec-ing-a-shel-command-in-pod/5354/5.
func execRemoteCommand(restCfg *rest.Config, clientset *kubernetes.Clientset, pod *coreapi.Pod, command []string) (string, string, error) {
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := clientset.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&coreapi.PodExecOptions{
			Command: command,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, kubescheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(restCfg, "POST", request.URL())
	if err != nil {
		return "", "", err
	}

	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", "", fmt.Errorf("%w Failed executing command %s on %v/%v", err, command, pod.Namespace, pod.Name)
	}

	// Return stdout, stderr.
	return buf.String(), errBuf.String(), nil
}
*/
