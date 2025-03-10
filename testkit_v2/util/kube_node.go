package integration

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apirtschema "k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
)

var (
	nodeGroupResource = apirtschema.GroupVersionResource{Group: "deckhouse.io", Version: "v1", Resource: "nodegroups"}
)

/*  Node  */

type NodeFilter struct {
	Name    any
	Os      any
	Kernel  any
	Kubelet any
}

type nodeType = coreapi.Node

func (f *NodeFilter) Apply(nodes []nodeType) (resp []nodeType) {
	for _, node := range nodes {
		if f.Name != nil && !CheckCondition(f.Name, node.ObjectMeta.Name) {
			continue
		}
		if f.Os != nil && !CheckCondition(f.Os, node.Status.NodeInfo.OSImage) {
			continue
		}
		if f.Kernel != nil && !CheckCondition(f.Kernel, node.Status.NodeInfo.KernelVersion) {
			continue
		}
		if f.Kubelet != nil && !CheckCondition(f.Kubelet, node.Status.NodeInfo.KubeletVersion) {
			continue
		}
		resp = append(resp, node)
	}
	return
}

func (clr *KCluster) ListNode(filters ...NodeFilter) ([]nodeType, error) {
	nodeList, err := (*clr.goClient).CoreV1().Nodes().List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Warnf("Can't get Nodes: %s", err.Error())
		return nil, err
	}

	resp := nodeList.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (clr *KCluster) MapLabelNodes(label any, filters ...NodeFilter) map[string][]string {
	resp := map[string][]string{}

	nodes, err := clr.ListNode(filters...)
	if err != nil {
		return nil
	}
	for lName, lFilter := range NodeRequired {
		if label != nil && !CheckCondition(label, lName) {
			continue
		}

		resp[lName] = []string{}
		for _, node := range lFilter.Apply(nodes) {
			resp[lName] = append(resp[lName], node.ObjectMeta.Name)
		}
	}

	return resp
}

/*  Node Group  */

func (clr *KCluster) ListNodeGroup() ([]unstructured.Unstructured, error) {
	ngs, err := clr.dyClient.Resource(nodeGroupResource).
		List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ngs.Items, err
}

func (clr *KCluster) CreateNodeGroup(data map[string]interface{}) error {
	name := data["metadata"].(map[string]interface{})["name"]
	obj := unstructured.Unstructured{}
	obj.SetUnstructuredContent(data)

	_, err := clr.dyClient.Resource(nodeGroupResource).
		Create(clr.ctx, &obj, metav1.CreateOptions{})
	if err == nil {
		Infof("NodeGroup %q created", name)
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		Infof("NodeGroup %q updating ...", name)
		content, err := obj.MarshalJSON()
		if err != nil {
			return err
		}
		_, err = clr.dyClient.Resource(nodeGroupResource).
			Patch(clr.ctx, obj.GetName(), apitypes.MergePatchType, content, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	return err
}

func (clr *KCluster) CreateNodeGroupStatic(name, role string, count int) error {
	ngObj := map[string]interface{}{
		"apiVersion": "deckhouse.io/v1",
		"kind":       "NodeGroup",
		"metadata": map[string]interface{}{
			"name": name,
		},
		"spec": map[string]interface{}{
			"nodeType": "Static",
			"staticInstances": map[string]interface{}{
				"count": count,
				"labelSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"node-role": role,
					},
				},
			},
		},
	}
	return clr.CreateNodeGroup(ngObj)
}

func (clr *KCluster) DeleteNodeGroup(name string) error {
	err := clr.dyClient.Resource(nodeGroupResource).Delete(clr.ctx, name, metav1.DeleteOptions{})
	if err == nil || apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

/*  Static Instance  */

func (clr *KCluster) ListStaticInstance() ([]StaticInstance, error) {
	sis := StaticInstanceList{}
	err := clr.rtClient.List(clr.ctx, &sis)
	if err != nil {
		return nil, err
	}

	return sis.Items, nil
}

func (clr *KCluster) CreateStaticInstance(name, role, ip, credentials string) error {
	si := &StaticInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role": role},
		},
		Spec: StaticInstanceSpec{
			Address:        ip,
			CredentialsRef: GetSSHCredentialsRef(credentials),
		},
	}

	if err := clr.rtClient.Create(clr.ctx, si); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}

		Errf("Can't create StaticInstance %s", name)
		return err
	}
	return nil
}

func (clr *KCluster) DeleteStaticInstance(name string) error {
	si := &StaticInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return clr.rtClient.Delete(clr.ctx, si)
}

/*  Static Node  */

func (clr *KCluster) AddStaticNodes(name, user string, ips []string) error {
	privSshKey, err := os.ReadFile(filepath.Join(KubePath, PrivKeyName))
	if err != nil {
		Errf("Read %s: %s", PrivKeyName, err.Error())
		return err
	}
	b64SshKey := base64.StdEncoding.EncodeToString(privSshKey)

	credentialName := name + "rsa"
	if err = clr.CreateOrUpdSSHCredential(credentialName, user, b64SshKey); err != nil {
		Errf("Create SSHCredential: %s", err.Error())
		return err
	}

	role := fmt.Sprintf("vm-%s-worker", name)
	for _, ip := range ips {
		siName := fmt.Sprintf("si-%s-%s", name, hashMd5(ip)[:8])
		err = clr.CreateStaticInstance(siName, role, ip, credentialName)
		if err != nil {
			Errf("Create StaticInstance %s: %s", siName, err.Error())
			return err
		}
	}

	if err = clr.CreateNodeGroupStatic(role, role, len(ips)); err != nil {
		Errf("Create NodeGroup: %s", err.Error())
		return err
	}

	return nil
}

/*  Pod  */

func (clr *KCluster) GetPod(nsName, pName string) (*coreapi.Pod, error) {
	pod, err := clr.goClient.CoreV1().Pods(nsName).Get(clr.ctx, pName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return pod, nil
	}
	return pod, err
}

func (clr *KCluster) ListPod(nsName string) ([]coreapi.Pod, error) {
	pods, err := clr.goClient.CoreV1().Pods(nsName).List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (clr *KCluster) CreatePod(nsName, pName string) error {
	if nsName == "" {
		nsName = TestNS
	}

	pod := coreapi.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pName,
			Namespace: nsName,
		},
		Spec: coreapi.PodSpec{},
	}
	if err := clr.rtClient.Create(clr.ctx, &pod); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}

		return err
	}

	return nil
}

func (clr *KCluster) DeletePod(nsName, pName string) error {
	if nsName == "" {
		nsName = TestNS
	}

	pod := coreapi.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pName,
			Namespace: nsName,
		},
	}
	if err := clr.rtClient.Delete(clr.ctx, &pod); err != nil {
		return err
	}

	return nil
}
