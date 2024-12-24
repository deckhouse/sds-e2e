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
	"k8s.io/apimachinery/pkg/api/resource"
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


/*  Filters  */

type Filter struct {  //map[string]any
	Name []string
	NotName []string
	Os []string
	NotOs []string
	Consumable string
}

func (f *Filter) checkName(name string) bool {
	if f.Name != nil {
		return slices.Contains(f.Name, name)
	}
	if f.NotName != nil {
		return !slices.Contains(f.NotName, name)
	}
	return true
}

func (f *Filter) checkConsumable(bd snc.BlockDevice) bool {
	if (f.Consumable == "true" && !bd.Status.Consumable) ||
	   (f.Consumable == "false" && bd.Status.Consumable) {
		return false
	}
	return true
}

func (f *Filter) checkOs(node coreapi.Node) bool {
		img := node.Status.NodeInfo.OSImage
		valid := true

		if f.Os != nil {
			valid = false
			for _, i := range f.Os {
				if strings.Contains(img, i) {
					valid = true
					break
				}
			}
		} else if f.NotOs != nil {
			for _, i := range f.NotOs {
				if strings.Contains(img, i) {
					valid = false
					break
				}
			}
		}

		return valid
}

/*  Logs  */

func initLog() {
	ctrlrt.SetLogger(logr.New(ctrlrtlog.NullLogSink{}))
}

func Infof(format string, v ...any) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("    \033[2m✎ ")
	log.Printf("\033[2m"+format+"\033[0m", v...)
}

func Warnf(format string, v ...any) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("    \033[93m🏱 ")
	log.Printf("\033[0;2m"+format+"\033[0m", v...)
}

func Errf(format string, v ...any) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("    \033[91m🕩 ")
	log.Printf("\033[0m"+format+"\033[0m", v...)
}

func Critf(format string, v ...any) {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("    \033[91;1;5m🕪 ")
	log.Printf("\033[0;91m"+format+"\033[0m", v...)
}

/*  Config  */

func NewRestConfig(configPath, clusterName string) (*rest.Config, error) {
	var cfg *rest.Config
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

/*  Kuber Client  */

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

/*  Kuber Cluster object  */

type KCluster struct {
	name string
	ctx context.Context
	rtClient *ctrlrtclient.Client
	goClient *kubernetes.Clientset
}

func InitKCluster(configPath, clusterName string) (*KCluster, error) {
	configPath = envConfigPath(configPath)
	clusterName	= envClusterName(clusterName)

	Infof("Init Client %s/%s", configPath, clusterName)
	rcl, err := NewKubeRTClient(configPath, clusterName)
    if err != nil {
		Critf("Can`t connect cluster %s", clusterName)
    	return nil, err
    }

	gcl, err := NewKubeGoClient(configPath, clusterName)
    if err != nil {
		Critf("Can`t connect cluster %s", clusterName)
    	return nil, err
    }

	ctr := KCluster{
		name: clusterName,
		ctx: context.Background(),
		rtClient: rcl,
		goClient: gcl,
	}

// TODO create test NameSpace

	return &ctr, nil
}

/*  Node  */

func (clr *KCluster) GetNodes(filter *Filter) (map[string]coreapi.Node, error) {
	resp := make(map[string]coreapi.Node)

	nodes, err := (*clr.goClient).CoreV1().Nodes().List(clr.ctx, metav1.ListOptions{})
    if err != nil {
		Errf("Can`t get Nodes (%s)", clr.name)
        return nil, err
    }

	for _, node := range nodes.Items {
		if filter != nil && !filter.checkOs(node) {
			continue
		}
		resp[node.ObjectMeta.Name] = node
	}

	return resp, nil
}

/*  Block Device  */

func (clr *KCluster) GetBDs(filter *Filter) (map[string]snc.BlockDevice, error)  {
	resp := make(map[string]snc.BlockDevice)

    bdList := &snc.BlockDeviceList{}
    err := (*clr.rtClient).List(clr.ctx, bdList)
    if err != nil {
		Errf("Can`t get BDs (%s)", clr.name)
        return nil, err
    }

	var validNodes map[string]coreapi.Node
	if filter != nil && (filter.Os != nil || filter.NotOs != nil) {
		validNodes, _ = clr.GetNodes(&Filter{Os: filter.Os, NotOs: filter.NotOs})
	}
	for _, bd := range bdList.Items {
		if filter != nil && !filter.checkConsumable(bd) {
			continue
		}
		if filter != nil && !filter.checkName(bd.Name) {
			continue
		}
		if validNodes != nil {
			if _, ok := validNodes[bd.Status.NodeName]; !ok {
				continue
			}
		}
		resp[bd.Name] = bd
	}

	return resp, nil
}

/*  LVM Volume Group  */

func (clr *KCluster) GetLVGs() ([]snc.LVMVolumeGroup, error) {
	lvgList := &snc.LVMVolumeGroupList{}
	err := (*clr.rtClient).List(clr.ctx, lvgList)
	if err != nil {
		Errf("Can`t get LVGs (%s)", clr.name)
        return nil, err
	}

	return lvgList.Items, nil
}

func (clr *KCluster) GetTestLVGs() (resp []snc.LVMVolumeGroup, err error) {
	lvgList, err := clr.GetLVGs()
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

func (clr *KCluster) CreateLVG(name, nodeName, bdName string) (*snc.LVMVolumeGroup, error) {
	lvmVolumeGroup := &snc.LVMVolumeGroup{
	    ObjectMeta: metav1.ObjectMeta{
	        Name: name,
	    },
	    Spec: snc.LVMVolumeGroupSpec{
	        ActualVGNameOnTheNode: name,
	        BlockDeviceSelector: &metav1.LabelSelector{
	            //MatchLabels: map[string]string{"kubernetes.io/metadata.name": bdName},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "kubernetes.io/metadata.name", Operator: metav1.LabelSelectorOpIn, Values: []string{bdName}},
				},
	        },
	        Type: "Local",
	        Local: snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
	    },
	}
	err := (*clr.rtClient).Create(clr.ctx, lvmVolumeGroup)
	if err != nil {
		Errf("Can`t create LVG %s (node %s, bd %s)", name, nodeName, bdName)
		return nil, err
	}
	return lvmVolumeGroup, nil
}

func (clr *KCluster) CreateTestLVG(nodeName, bdName string) (*snc.LVMVolumeGroup, error) {
	name := "e2e-lvg-" + nodeName[len(nodeName)-1:] + "-" + bdName[len(bdName)-3:]
	return clr.CreateLVG(name, nodeName, bdName)
}

func (clr *KCluster) UpdateLVG(lvg *snc.LVMVolumeGroup) error {
	err := (*clr.rtClient).Update(clr.ctx, lvg)
	if err != nil {
		Errf("Can`t update LVG %s", lvg.Name)
		return err
    }

	return nil
}

func (clr *KCluster) DelTestLVG() error {
	lvgList, _ := clr.GetTestLVGs()

	for _, lvg := range lvgList {
		if err := (*clr.rtClient).Delete(clr.ctx, &lvg); err != nil {
			return err
		}
		Infof("LVG deleted: %s", lvg.Name)
	}

	return nil
}

/*  Storage Class  */

func (clr *KCluster) CreateSC(name string) (*storapi.StorageClass, error) {
	lvmType := "Thick"
	lvmVolGroups := "- name: vg-w1\n- name: vg-w2"

	volBindingMode := storapi.VolumeBindingImmediate
	//volBindingMode := storapi.VolumeBindingWaitForFirstConsumer

    reclaimPolicy := coreapi.PersistentVolumeReclaimDelete
	//reclaimPolicy := coreapi.PersistentVolumeReclaimRetain

	volExpansion := true

    sc := &storapi.StorageClass{
        TypeMeta: metav1.TypeMeta{
            Kind: "StorageClass",
            APIVersion: "storage.k8s.io/v1",
        },
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: "default",
        },
        Provisioner: "lvm.csi.storage.deckhouse.io",
        Parameters: map[string]string{
            "lvm.csi.storage.deckhouse.io/lvm-type":            lvmType,
            "lvm.csi.storage.deckhouse.io/volume-binding-mode": string(volBindingMode),
            "lvm.csi.storage.deckhouse.io/lvm-volume-groups":   lvmVolGroups,
        },
        ReclaimPolicy:        &reclaimPolicy,
        MountOptions:         nil,
        AllowVolumeExpansion: &volExpansion,
        VolumeBindingMode:    &volBindingMode,
    }

    if err := (*clr.rtClient).Create(clr.ctx, sc); err != nil {
		Errf("Can`t create SC %s", sc.Name)
        return nil, err
    }
    return sc, nil
}

/*  Persistent Volume Claims  */

func (clr *KCluster) GetPVC(nsName string) ([]coreapi.PersistentVolumeClaim, error) {
    pvcList, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).List(clr.ctx, metav1.ListOptions{})
    //pvc, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).Get(clr.ctx, pvcName, metav1.GetOptions{})

//    pvcList := coreapi.PersistentVolumeClaimList{}
//    opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})
//    err := (*clr.rtClient).List(clr.ctx, &pvcList, opts)
    if err != nil {
		Errf("Can`t get PVCs %s", nsName)
		return nil, err
    }

	return pvcList.Items, nil
}

func (clr *KCluster) GetTestPVC() ([]coreapi.PersistentVolumeClaim, error) {
    return clr.GetPVC("d8-upmeter")
    return clr.GetPVC("sds-replicated-volume-e2e-test")
}


//======================================
const (
    PersistentVolumeClaimKind       = "PersistentVolumeClaim"
    PersistentVolumeClaimAPIVersion = "v1"
    WaitIntervalPVC                 = 1
    WaitIterationCountPVC           = 10
    DeletedStatusPVC                = "Deleted"
	//NameSpace       = "sds-local-volume"
)

func (clr *KCluster) CreatePVC(name, scName, size string) (*coreapi.PersistentVolumeClaim, error) {
	resourceList := make(map[coreapi.ResourceName]resource.Quantity)
	sizeStorage, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	resourceList[coreapi.ResourceStorage] = sizeStorage
	volMode := coreapi.PersistentVolumeFilesystem
	//volMode := coreapi.PersistentVolumeBlock

	pvc := &coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       PersistentVolumeClaimKind,
			APIVersion: PersistentVolumeClaimAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNS,
			//Namespace: NameSpace,
		},
		Spec: coreapi.PersistentVolumeClaimSpec{
			StorageClassName: &scName,
			AccessModes: []coreapi.PersistentVolumeAccessMode{
				coreapi.ReadWriteOnce,
			},
			Resources: coreapi.VolumeResourceRequirements{
				Requests: resourceList,
			},
			VolumeMode: &volMode,
		},
	}

	err = (*clr.rtClient).Create(clr.ctx, pvc)
	if err != nil {
		return nil, err
		//return coreapi.PersistentVolumeClaim{}, err
	}
	return pvc, nil
}

/* TODO
func DeletePVC(ctx context.Context, cl client.Client, name string) error {
	pvc := coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       PersistentVolumeClaimKind,
			APIVersion: PersistentVolumeClaimAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NameSpace,
		},
	}

	err := cl.Delete(ctx, &pvc)
	if err != nil {
		return err
	}
	return nil
}

func WaitPVCStatus(ctx context.Context, cl client.Client, name string) (string, error) {
	pvc := coreapi.PersistentVolumeClaim{}
	for i := 0; i < WaitIterationCountPVC; i++ {
		err := cl.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: NameSpace,
		}, &pvc)
		if err != nil {
			//if kerrors.IsNotFound(err) {
			//	return "", err
			//}
		}
		fmt.Printf("pvc %s...\n", pvc.Status.Phase)
		if pvc.Status.Phase == coreapi.ClaimBound {
			return string(pvc.Status.Phase), nil
		}

		if len(pvc.Status.Phase) == 0 {
			return DeletedStatusPVC, nil
		}

		time.Sleep(WaitIntervalPVC * time.Second)
	}
	return "", errors.New(fmt.Sprintf("the waiting time %d or the pvc to be ready has expired",
		WaitIntervalPVC*WaitIterationCountPVC))
}

func WaitDeletePVC(ctx context.Context, cl client.Client, name string) (string, error) {
	pod := coreapi.Pod{}
	for i := 0; i < WaitIterationCountPVC; i++ {
		time.Sleep(WaitIntervalPVC * time.Second)
		err := cl.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: NameSpace,
		}, &pod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return DeletedStatusPVC, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("the waiting time %d for the pod to be ready has expired",
		WaitIterationCountPVC*WaitIntervalPVC))
}

func EditSizePVC(ctx context.Context, cl client.Client, name, newSize string) error {
	pvc := coreapi.PersistentVolumeClaim{}
	err := cl.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: NameSpace,
	}, &pvc)
	if err != nil {
		return err
	}

	resourceList := make(map[coreapi.ResourceName]resource.Quantity)
	newPVCSize, err := resource.ParseQuantity(newSize)
	if err != nil {
		return err
	}

	resourceList[coreapi.ResourceStorage] = newPVCSize
	pvc.Spec.Resources.Requests = resourceList

	err = cl.Update(ctx, &pvc)
	if err != nil {
		return err
	}
	return nil
}
*/

func (clr *KCluster) UpdatePVC(pvc *coreapi.PersistentVolumeClaim) error {
//	nsName := pvc.Namespace
//	pvc, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).Update(clr.ctx, pvc, metav1.UpdateOptions{})
	err := (*clr.rtClient).Update(clr.ctx, pvc)
	if err != nil {
		Errf("Can`t update PVC %s", pvc.Name)
		return err
    }

	return nil
}

/*  Virtual Disk  */

func (clr *KCluster) GetVMD(nsName, vmdName string) ([]virt.VirtualDisk, error) {
	vmds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})

	err := (*clr.rtClient).List(clr.ctx, &vmds, opts)
	if err != nil {
		Errf("Can`t get VMDs (%s, %s)", nsName, vmdName)
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

func (clr *KCluster) UpdateVMD(vmd *virt.VirtualDisk) error {
	return (*clr.rtClient).Update(clr.ctx, vmd)
}

/*  Exec Cmd  */
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
