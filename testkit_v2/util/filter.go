package integration

import (
	"regexp"
	"strings"

	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	coreapi "k8s.io/api/core/v1"
)


type Cond struct {
	// Check conditions
	In          []string
	NotIn       []string
	Contains    []string
	NotContains []string
	Regexp      []string
	NotRegexp   []string
}

func (c *Cond) isValid(arg interface{}) bool {
	switch t := arg.(type) {
	case string:
		val := arg.(string)

		for _, v := range c.NotIn {
			if val == v {
				return false
			}
		}
		for _, v := range c.NotContains {
			if strings.Contains(val, v) {
				return false
			}
		}
		for _, v := range c.NotRegexp {
			match, err := regexp.MatchString(v, val)
			if err != nil || match {
				return false
			}
		}

		for _, v := range c.In {
			if val == v {
				return true
			}
		}
		for _, v := range c.Contains {
			if strings.Contains(val, v) {
				return true
			}
		}
		for _, v := range c.Regexp {
			match, err := regexp.MatchString(v, val)
			if err == nil && match {
				return true
			}
		}
	case bool:
		val := arg.(bool)

		if len(c.In) == 0 {
			return true
		}
		if c.In[0] == "true" && val || c.In[0] == "false" && !val {
			return true
		}
		return false
	default:
		Errf("Invalid compare type: %T", t)
		return false
	}

	return len(c.In) + len(c.Contains) + len(c.Regexp) == 0
}

type NodeFilter struct {
	Name       Cond
	NameSpace  Cond
	Os         Cond
	Kernel     Cond
	Kubelet    Cond
	NodeGroup  Cond
}

func (f *NodeFilter) Check(node coreapi.Node) bool {
	if !f.Name.isValid(node.ObjectMeta.Name) {
		return false
	}
	//if !f.NameSpace.isValid() {
	//Not implemented
	//}
	if !f.Os.isValid(node.Status.NodeInfo.OSImage) {
		return false
	}
	if !f.Kernel.isValid(node.Status.NodeInfo.KernelVersion) {
		return false
	}
	if !f.Kubelet.isValid(node.Status.NodeInfo.KubeletVersion) {
		return false
	}
	//if !f.NodeGroup.isValid() {
	//Not implemented
	//}

	return true
}

func (f *NodeFilter) Intersec(val []string) bool {
	return intersec(val, f.NodeGroup.In, f.NodeGroup.NotIn)
}

type BdFilter struct {
	Name       Cond
	Node       Cond
	Consumable Cond
}

func (f *BdFilter) Check(bd snc.BlockDevice) bool {
	if !f.Name.isValid(bd.Name) {
		return false
	}
	if !f.Node.isValid(bd.Status.NodeName) {
		return false
	}
	if !f.Consumable.isValid(bd.Status.Consumable) {
		return false
	}

	return true
}

type NsFilter struct {
	Name      Cond
	NameSpace Cond
}

func (f *NsFilter) Check(ns coreapi.Namespace) bool {
	if !f.Name.isValid(ns.Name) {
		return false
	}
	if !f.NameSpace.isValid(ns.Name) {
		return false
	}

	return true
}

type LvgFilter struct {
	Name       Cond
	Node       Cond
}

func (f *LvgFilter) Check(lvg snc.LVMVolumeGroup) bool {
	if !f.Name.isValid(lvg.Name) {
		return false
	}
	if len(lvg.Status.Nodes) > 0 && !f.Node.isValid(lvg.Status.Nodes[0].Name) {
		return false
	}
	if !f.Node.isValid("No node for LVG object") {
		return false
	}

	return true
}


// TODO REMOVE OLD VERSION

type Filter struct {
	Name         []string
	NotName      []string
	Os           []string
	NotOs        []string
	Node         []string
	NotNode      []string
	NodeGroup    []string
	NotNodeGroup []string
	Consumable   string
	Kernel       []string
	NotKernel    []string
	Kubelet      []string
	NotKubelet   []string
	NS           []string
	NotNS        []string
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

func (f *Filter) checkNode(node coreapi.Node) bool {
	if !f.match(node.ObjectMeta.Name, f.Node, f.NotNode) {
		return false
	}

	// check name
	if !f.like(node.ObjectMeta.Name, f.Name, f.NotName) {
		return false
	}

	// check os
	if !f.checkOs(node) {
		return false
	}

	// check core
	if !f.like(node.Status.NodeInfo.KernelVersion, f.Kernel, f.NotKernel) {
		return false
	}

	// check kubelet
	if !f.like(node.Status.NodeInfo.KubeletVersion, f.Kubelet, f.NotKubelet) {
		return false
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

func (f *Filter) match(val string, in []string, notIn []string) bool {
	return match(val, in, notIn)
}
func match(val string, in []string, notIn []string) bool {
	for _, v := range notIn {
		if val == v {
			return false
		}
		if strings.HasPrefix(v, "~") {
			match, err := regexp.MatchString(v[1:], val)
			if err == nil && match {
				return false
			}
		}
	}
	if in == nil {
		return true
	}
	for _, v := range in {
		if val == v {
			return true
		}
		if strings.HasPrefix(v, "~") {
			match, err := regexp.MatchString(v[1:], val)
			if err == nil && match {
				return true
			}
		}
	}
	return false
}

func (f *Filter) like(val string, in []string, notIn []string) bool {
	return like(val, in, notIn)
}
func like(val string, in []string, notIn []string) bool {
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
	return intersec(val, in, notIn)
}
func intersec(val []string, in []string, notIn []string) bool {
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

