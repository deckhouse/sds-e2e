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
type OSType struct {
	Name          string
	ImageURL      string
	KernelVersion string
}

var (
	OSTypeMap = map[string]OSType{
		"Ubuntu 22.04 6.2.0-39-generic": {
			ImageURL:      "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
			KernelVersion: "6.2.0-39-generic",
		},
		"Ubuntu 24.04 6.8.0-53-generic": {
			ImageURL:      "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
			KernelVersion: "6.8.0-53-generic",
		},
	}
)

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodSSHKey  AuthMethod = "ssh-key"
	AuthMethodSSHPass AuthMethod = "ssh-password"
)

// NodeAuth contains authentication information for a node
type NodeAuth struct {
	Method   AuthMethod `yaml:"method"`
	User     string     `yaml:"user"`
	SSHKey   string     `yaml:"sshKey"`   // Public key (value like "ssh-rsa ...", path to .pub file, or empty for default ~/.ssh/id_rsa.pub)
	Password string     `yaml:"password"` // Password (if using password auth)
	// Internal: resolved private key path for SSH authentication (not in YAML)
	privateKeyPath string
}

// ClusterNode defines a single node in the cluster
type ClusterNode struct {
	Hostname  string      `yaml:"hostname"`
	IPAddress string      `yaml:"ipAddress,omitempty"` // Required for bare-metal, optional for VM
	OSType    OSType      `yaml:"osType,omitempty"`    // Required for VM, optional for bare-metal
	HostType  HostType    `yaml:"hostType"`
	Role      ClusterRole `yaml:"role"`
	Auth      NodeAuth    `yaml:"auth"`
	// VM-specific fields (only used when HostType == HostTypeVM)
	CPU      int `yaml:"cpu"`      // Required for VM
	RAM      int `yaml:"ram"`      // Required for VM, in GB
	DiskSize int `yaml:"diskSize"` // Required for VM, in GB
	// Image field removed - osType is used as image definition
	// Bare-metal specific fields
	Prepared bool `yaml:"prepared,omitempty"` // Whether the node is already prepared for DKP installation
}

// ClusterDefinition defines the complete cluster configuration
type ClusterDefinition struct {
	Masters []ClusterNode `yaml:"masters"`
	Workers []ClusterNode `yaml:"workers"`
	Setup   *ClusterNode  `yaml:"setup,omitempty"` // Bootstrap node (can be nil, will use master for VM clusters or first worker for bare-metal if not set)
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
