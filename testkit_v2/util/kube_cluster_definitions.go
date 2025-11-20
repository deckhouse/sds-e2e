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

import "time"

// HostType represents the type of host (VM or bare-metal)
type HostType string

const (
	HostTypeVM        HostType = "vm"
	HostTypeBareMetal HostType = "bare-metal"
)

// ClusterRole represents the role of a node in the cluster
type ClusterRole string

const (
	ClusterRoleMaster ClusterRole = "master"
	ClusterRoleWorker ClusterRole = "worker"
	ClusterRoleSetup  ClusterRole = "setup" // Bootstrap node for DKP installation
)

// OSType represents the operating system type
type OSType string

const (
	OSTypeUbuntu22 OSType = "Ubuntu_22"
	OSTypeUbuntu24 OSType = "Ubuntu_24"
	OSTypeDebian11 OSType = "Debian_11"
	OSTypeAstra173 OSType = "Astra_173"
	OSTypeAstra175 OSType = "Astra_175"
	OSTypeAstra181 OSType = "Astra_181"
	OSTypeRedOS73  OSType = "RedOS_7_3"
	OSTypeRedOS8   OSType = "RedOS_8"
	OSTypeAlt10    OSType = "Alt_10"
)

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodSSHKey  AuthMethod = "ssh-key"
	AuthMethodSSHPass AuthMethod = "ssh-password"
)

// NodeAuth contains authentication information for a node
type NodeAuth struct {
	Method   AuthMethod
	User     string
	SSHKey   string // Path to SSH private key
	Password string // Password (if using password auth)
}

// ClusterNode defines a single node in the cluster
type ClusterNode struct {
	Hostname  string
	IPAddress string
	OSType    OSType
	HostType  HostType
	Role      ClusterRole
	Auth      NodeAuth
	// VM-specific fields (only used when HostType == HostTypeVM)
	CPU      int
	RAM      int // in GB
	DiskSize int // in GB
	Image    string
	// Bare-metal specific fields
	Prepared bool // Whether the node is already prepared for DKP installation
}

// ClusterDefinition defines the complete cluster configuration
type ClusterDefinition struct {
	Masters []ClusterNode
	Workers []ClusterNode
	Setup   *ClusterNode // Bootstrap node (can be nil, will use first worker if not set)
}

// ModuleConfig defines a Deckhouse module configuration
type ModuleConfig struct {
	Name         string
	Version      int
	Enabled      bool
	Settings     map[string]any
	Dependencies []string // Names of modules that must be enabled before this one
}

// DKPClusterConfig defines the Deckhouse Kubernetes Platform cluster configuration
type DKPClusterConfig struct {
	ClusterDefinition ClusterDefinition
	KubernetesVersion string
	PodSubnetCIDR     string
	ServiceSubnetCIDR string
	ClusterDomain     string
	LicenseKey        string
	RegistryRepo      string
}

const (
	HostReadyTimeout    = 10 * time.Minute // Timeout for hosts to be ready
	DKPDeployTimeout    = 30 * time.Minute // Timeout for DKP deployment
	ModuleDeployTimeout = 10 * time.Minute // Timeout for module deployment
)
