package integration

import (
	"strings"
	"slices"

	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	coreapi "k8s.io/api/core/v1"
)


type Filter struct {
	Name []string
	NotName []string
	Os []string
	NotOs []string
	Node []string
	NotNode []string
	NodeGroup []string
	NotNodeGroup []string
	Consumable string
	Kernel []string
	NotKernel []string
	Kubelet []string
	NotKubelet []string
}

func (f *Filter) match(val string, in []string, notIn []string) bool {
	if in != nil {
		if !slices.Contains(in, val) {
			return false
		}
	}
	if notIn != nil {
		return !slices.Contains(notIn, val)
	}

	return true
}

func (f *Filter) like(val string, in []string, notIn []string) bool {
	for _, v := range notIn {
		if strings.Contains(val, v) {
			return false
		}
	}
	if in != nil {
		for _, v := range in {
			if strings.Contains(val, v) {
				return true
			}
		}
		return false
	}

	return true
}

func (f *Filter) intersec(val []string, in []string, notIn []string) bool {
	if val == nil {
		return true
	}

	set := map[string]interface{}{}
	for _, v := range val {
		set[v] = nil
	}

	for _, v := range notIn {
		delete(set, v)
	}
	if in != nil {
		for _, v := range in {
			if _, ok := set[v]; ok {
				return true
			}
		}
		return false
	}

	return len(set) > 0 
}

func (f *Filter) checkName(name string) bool {
	return f.match(name, f.Name, f.NotName)
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

//func (f *Filter) checkNode(nodeName string) bool {
//	return f.match(nodeName, f.Node, f.NotNode)
//	//valid := true
//
//	//if f.Node != nil {
//	//	valid = false
//	//	for _, n := range f.Node {
//	//		if nodeName == n {
//	//			valid = true
//	//			break
//	//		}
//	//	}
//	//}
//	//if f.NotNode != nil {
//	//	for _, n := range f.NotNode {
//	//		if nodeName == n {
//	//			valid = false
//	//			break
//	//		}
//	//	}
//	//}
//
//	//return valid
//}

func (f *Filter) checkNode(node coreapi.Node) bool {
	if !f.match(node.ObjectMeta.Name, f.Node, f.NotNode) {
		return false
	}

    // check os
	if !f.checkOs(node) {
		return false
	}

	// check core
	kernel := node.Status.NodeInfo.KernelVersion
	if !f.like(kernel, f.Kernel, f.NotKernel) {
		return false
	}

	// TODO check Kubelet
	//kubelet := node.Status.NodeInfo.KubeletVersion

	return true
}

func (f *Filter) checkConsumable(bd snc.BlockDevice) bool {
	if (f.Consumable == "true" && !bd.Status.Consumable) ||
	   (f.Consumable == "false" && bd.Status.Consumable) {
		return false
	}
	return true
}
