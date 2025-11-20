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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1alpha1nfs "github.com/deckhouse/csi-nfs/api/v1alpha1"
	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureHostsReady ensures that all hosts (VMs or bare-metal) are ready for DKP deployment
func EnsureHostsReady(hvCluster *KCluster, definition ClusterDefinition) error {
	Infof("Ensuring hosts are ready for DKP deployment")

	var allNodes []ClusterNode
	allNodes = append(allNodes, definition.Masters...)
	allNodes = append(allNodes, definition.Workers...)
	if definition.Setup != nil {
		allNodes = append(allNodes, *definition.Setup)
	}

	// Process VMs
	for _, node := range allNodes {
		switch node.HostType {
		case HostTypeVM:
			if err := ensureVMReady(hvCluster, node); err != nil {
				return fmt.Errorf("VM %s (%s) is not ready: %w", node.Hostname, node.IPAddress, err)
			}
		case HostTypeBareMetal:
			if err := ensureBareMetalReady(node); err != nil {
				return fmt.Errorf("bare-metal node %s (%s) is not ready: %w", node.Hostname, node.IPAddress, err)
			}
		}
	}

	Infof("All hosts are ready")
	return nil
}

// ensureVMReady ensures a VM is deployed and accessible
func ensureVMReady(hvCluster *KCluster, node ClusterNode) error {
	Infof("Checking VM %s (%s)", node.Hostname, node.IPAddress)

	nsName := TestNS

	// Check if VM exists
	vmList, err := hvCluster.ListVM(VmFilter{NameSpace: nsName, Name: node.Hostname})
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	// Create VM if it doesn't exist
	if len(vmList) == 0 {
		Infof("Creating VM %s", node.Hostname)

		// For VM clusters, use the same logic as main branch:
		// Generate keys in KubePath and use the public key for VM creation
		// This ensures the private key is available in KubePath for SSH access
		sshPubKeyString := CheckAndGetSSHKeys(KubePath, PrivKeyName, PubKeyName)

		// Use osType as image definition (image field is deprecated)
		image := string(node.OSType)

		err = hvCluster.CreateVM(
			nsName,
			node.Hostname,
			node.IPAddress,
			node.CPU,
			node.RAM,
			HvStorageClass,
			image,
			sshPubKeyString,
			node.DiskSize,
		)
		if err != nil {
			return fmt.Errorf("failed to create VM: %w", err)
		}
	}

	// Wait for VM to be running
	if err := RetrySec(int(HostReadyTimeout.Seconds()), func() error {
		vmList, err := hvCluster.ListVM(VmFilter{NameSpace: nsName, Name: node.Hostname, Phase: string(virt.MachineRunning)})
		if err != nil {
			return err
		}
		if len(vmList) == 0 {
			return fmt.Errorf("VM %s is not running yet", node.Hostname)
		}
		// Update IP address if it was not set
		if node.IPAddress == "" && len(vmList) > 0 {
			node.IPAddress = vmList[0].Status.IPAddress
		}
		Debugf("VM %s is running with IP %s", node.Hostname, node.IPAddress)
		return nil
	}); err != nil {
		return fmt.Errorf("VM %s failed to become ready: %w", node.Hostname, err)
	}

	// Verify SSH access
	if err := verifySSHAccess(node); err != nil {
		return fmt.Errorf("VM %s SSH access failed: %w", node.Hostname, err)
	}

	Infof("VM %s is ready and accessible", node.Hostname)
	return nil
}

// ensureBareMetalReady ensures a bare-metal node is accessible and prepared
func ensureBareMetalReady(node ClusterNode) error {
	Infof("Checking bare-metal node %s (%s)", node.Hostname, node.IPAddress)

	// Verify SSH access
	if err := verifySSHAccess(node); err != nil {
		return fmt.Errorf("bare-metal node %s SSH access failed: %w", node.Hostname, err)
	}

	// If node is not marked as prepared, check and install prerequisites
	if !node.Prepared {
		Infof("Preparing bare-metal node %s for DKP installation", node.Hostname)
		if err := prepareBareMetalNode(node); err != nil {
			return fmt.Errorf("failed to prepare bare-metal node %s: %w", node.Hostname, err)
		}
	} else {
		Infof("Bare-metal node %s is already prepared", node.Hostname)
		// Verify prerequisites are installed
		if err := verifyBareMetalPrerequisites(node); err != nil {
			return fmt.Errorf("bare-metal node %s prerequisites check failed: %w", node.Hostname, err)
		}
	}

	Infof("Bare-metal node %s is ready", node.Hostname)
	return nil
}

// verifySSHAccess verifies SSH access to a node
func verifySSHAccess(node ClusterNode) error {
	var keyPath string

	// For VM clusters, use the key from KubePath (same as main branch)
	// For bare-metal clusters, use the resolved private key path
	if node.HostType == HostTypeVM {
		// For VMs, use the private key from KubePath (generated by CheckAndGetSSHKeys)
		keyPath = filepath.Join(KubePath, PrivKeyName)
	} else {
		// For bare-metal, use the resolved private key path
		keyPath = node.Auth.privateKeyPath
		if keyPath == "" {
			keyPath = NestedSshKey
		}
	}

	user := node.Auth.User
	if user == "" {
		user = NestedSshUser
	}

	var client sshClient
	if node.HostType == HostTypeVM {
		// For VMs, we need to use the hypervisor SSH client with forwarding
		client = HvSshClient.GetFwdClient(user, node.IPAddress+":22", keyPath)
	} else {
		// For bare-metal, direct SSH connection
		client = GetSshClient(user, node.IPAddress+":22", keyPath)
	}
	defer client.Close()

	// Test SSH connection
	out, err := client.Exec("echo 'SSH connection successful'")
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w\nOutput: %s", err, out)
	}

	Debugf("SSH access verified for %s", node.Hostname)
	return nil
}

// prepareBareMetalNode prepares a bare-metal node for DKP installation
func prepareBareMetalNode(node ClusterNode) error {
	// For bare-metal, use the resolved private key path (from public key resolution)
	keyPath := node.Auth.privateKeyPath
	if keyPath == "" {
		keyPath = NestedSshKey
	}

	user := node.Auth.User
	if user == "" {
		user = NestedSshUser
	}

	client := GetSshClient(user, node.IPAddress+":22", keyPath)
	defer client.Close()

	// Ensure Docker is installed
	if err := ensureDockerInstalled(client); err != nil {
		return fmt.Errorf("failed to ensure Docker is installed: %w", err)
	}

	Infof("Bare-metal node %s is prepared", node.Hostname)
	return nil
}

// verifyBareMetalPrerequisites verifies that prerequisites are installed on bare-metal node
func verifyBareMetalPrerequisites(node ClusterNode) error {
	// For bare-metal, use the resolved private key path (from public key resolution)
	keyPath := node.Auth.privateKeyPath
	if keyPath == "" {
		keyPath = NestedSshKey
	}

	user := node.Auth.User
	if user == "" {
		user = NestedSshUser
	}

	client := GetSshClient(user, node.IPAddress+":22", keyPath)
	defer client.Close()

	// Check Docker
	out, err := client.Exec("docker --version")
	if err != nil || !strings.Contains(out, "Docker version") {
		return fmt.Errorf("docker is not installed or not accessible: %w", err)
	}

	Debugf("Prerequisites verified for %s", node.Hostname)
	return nil
}

// EnsureDKPClusterReady ensures that DKP is deployed on the cluster
func EnsureDKPClusterReady(config DKPClusterConfig, hvCluster *KCluster) (*KCluster, error) {
	Infof("Ensuring DKP cluster is ready")

	// Identify master and setup nodes
	if len(config.ClusterDefinition.Masters) == 0 {
		return nil, fmt.Errorf("at least one master node is required")
	}

	masterNode := config.ClusterDefinition.Masters[0]
	setupNode := config.ClusterDefinition.Setup

	// If setup node is not set, determine based on cluster type
	if setupNode == nil {
		// For VM clusters, use master as setup node
		// For bare-metal clusters, prefer first worker if available, otherwise use master
		if masterNode.HostType == HostTypeVM {
			setupNode = &masterNode
		} else if len(config.ClusterDefinition.Workers) > 0 {
			setupNode = &config.ClusterDefinition.Workers[0]
		} else {
			setupNode = &masterNode
		}
	}

	// Check if Deckhouse is already installed
	// For setup node: use public key resolution logic (find private key from public key)
	// This applies to both VM and bare-metal setup nodes
	var keyPath string
	if setupNode.HostType == HostTypeVM {
		// For VM setup node, use the key from KubePath (generated by CheckAndGetSSHKeys)
		keyPath = filepath.Join(KubePath, PrivKeyName)
	} else {
		// For bare-metal setup node, use the resolved private key path
		keyPath = setupNode.Auth.privateKeyPath
		if keyPath == "" {
			keyPath = NestedSshKey
		}
	}

	user := setupNode.Auth.User
	if user == "" {
		user = NestedSshUser
	}

	var setupClient sshClient
	if setupNode.HostType == HostTypeVM {
		setupClient = HvSshClient.GetFwdClient(user, setupNode.IPAddress+":22", keyPath)
	} else {
		setupClient = GetSshClient(user, setupNode.IPAddress+":22", keyPath)
	}
	defer setupClient.Close()

	// Check if Deckhouse is installed
	if !checkDeckhouseInstalled() {
		Infof("Deploying Deckhouse on the cluster")
		if err := deployDKP(setupClient, masterNode); err != nil {
			return nil, fmt.Errorf("failed to deploy DKP: %w", err)
		}
	} else {
		Infof("Deckhouse is already installed")
	}

	// Set up NestedSshClient for kubeconfig retrieval
	// For master node: use public key resolution logic (find private key from public key)
	var masterKeyPath string
	if masterNode.HostType == HostTypeVM {
		// For VM master node, use the key from KubePath (generated by CheckAndGetSSHKeys)
		masterKeyPath = filepath.Join(KubePath, PrivKeyName)
	} else {
		// For bare-metal master node, use the resolved private key path
		masterKeyPath = masterNode.Auth.privateKeyPath
		if masterKeyPath == "" {
			masterKeyPath = NestedSshKey
		}
	}
	masterUser := masterNode.Auth.User
	if masterUser == "" {
		masterUser = NestedSshUser
	}

	if masterNode.HostType == HostTypeVM {
		NestedSshClient = HvSshClient.GetFwdClient(masterUser, masterNode.IPAddress+":22", masterKeyPath)
		go NestedSshClient.NewTunnel("127.0.0.1:"+NestedK8sPort, masterNode.IPAddress+":"+NestedK8sPort)
	} else {
		NestedSshClient = GetSshClient(masterUser, masterNode.IPAddress+":22", masterKeyPath)
	}

	// Get kubeconfig
	if err := getKubeconfig(); err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Initialize cluster client
	cluster, err := InitKCluster("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cluster client: %w", err)
	}

	// Wait for nodes to be ready
	expectedNodeCount := len(config.ClusterDefinition.Masters) + len(config.ClusterDefinition.Workers)
	if err := ensureNodesReady(cluster, expectedNodeCount); err != nil {
		return nil, fmt.Errorf("nodes are not ready: %w", err)
	}

	// Add worker nodes if they are VMs
	if len(config.ClusterDefinition.Workers) > 0 {
		workerIPs := make([]string, 0, len(config.ClusterDefinition.Workers))
		for _, worker := range config.ClusterDefinition.Workers {
			if worker.HostType == HostTypeVM {
				workerIPs = append(workerIPs, worker.IPAddress)
			}
		}

		if len(workerIPs) > 0 {
			Infof("Adding %d worker nodes to the cluster", len(workerIPs))
			if err := cluster.AddStaticNodes("e2e", user, workerIPs); err != nil {
				return nil, fmt.Errorf("failed to add static nodes: %w", err)
			}

			// Wait for worker nodes to be ready
			if err := ensureNodesReady(cluster, expectedNodeCount); err != nil {
				return nil, fmt.Errorf("worker nodes are not ready: %w", err)
			}
		}
	}

	Infof("DKP cluster is ready")
	return cluster, nil
}

// deployDKP deploys Deckhouse Kubernetes Platform
func deployDKP(client sshClient, masterNode ClusterNode) error {
	// Generate config files
	mkConfig()
	mkResources()

	// Upload bootstrap files
	if err := uploadBootstrapFiles(client); err != nil {
		return fmt.Errorf("failed to upload bootstrap files: %w", err)
	}

	// Install Deckhouse
	if err := installVmDh(client, masterNode.IPAddress); err != nil {
		return fmt.Errorf("failed to install Deckhouse: %w", err)
	}

	return nil
}

// EnsureModulesReady ensures that all required modules are deployed and ready
func EnsureModulesReady(cluster *KCluster, modules []ModuleConfig) error {
	Infof("Ensuring modules are ready")

	// Sort modules by dependencies
	sortedModules, err := sortModulesByDependencies(modules)
	if err != nil {
		return fmt.Errorf("failed to sort modules by dependencies: %w", err)
	}

	// Enable modules in order
	for _, module := range sortedModules {
		Infof("Ensuring module %s is enabled", module.Name)
		if err := ensureModuleEnabled(cluster, module); err != nil {
			return fmt.Errorf("failed to enable module %s: %w", module.Name, err)
		}
	}

	// Wait for modules to be ready
	for _, module := range sortedModules {
		Infof("Waiting for module %s to be ready", module.Name)
		if err := waitForModuleReady(cluster, module); err != nil {
			return fmt.Errorf("module %s is not ready: %w", module.Name, err)
		}
	}

	Infof("All modules are ready")
	return nil
}

// sortModulesByDependencies sorts modules by their dependencies
func sortModulesByDependencies(modules []ModuleConfig) ([]ModuleConfig, error) {
	// Create a map of module names to modules
	moduleMap := make(map[string]ModuleConfig)
	for _, module := range modules {
		moduleMap[module.Name] = module
	}

	// Topological sort
	var sorted []ModuleConfig
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving module %s", name)
		}
		if visited[name] {
			return nil
		}

		visiting[name] = true
		module, exists := moduleMap[name]
		if !exists {
			return fmt.Errorf("module %s not found", name)
		}

		for _, dep := range module.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		sorted = append(sorted, module)
		return nil
	}

	for _, module := range modules {
		if !visited[module.Name] {
			if err := visit(module.Name); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}

// ensureModuleEnabled ensures a module is enabled in the cluster
func ensureModuleEnabled(cluster *KCluster, module ModuleConfig) error {
	moduleConfig := &v1alpha1nfs.ModuleConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: module.Name,
		},
		Spec: v1alpha1nfs.ModuleConfigSpec{
			Version:  module.Version,
			Enabled:  ptr.To(module.Enabled),
			Settings: module.Settings,
		},
	}

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, moduleConfig)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create module config: %w", err)
		}
		// Module already exists, update it
		existingModuleConfig := &v1alpha1nfs.ModuleConfig{}
		err = cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{Name: module.Name}, existingModuleConfig)
		if err != nil {
			return fmt.Errorf("failed to get existing module config: %w", err)
		}
		moduleConfig.ResourceVersion = existingModuleConfig.ResourceVersion
		err = cluster.controllerRuntimeClient.Update(cluster.ctx, moduleConfig)
		if err != nil {
			return fmt.Errorf("failed to update module config: %w", err)
		}
	}

	return nil
}

// waitForModuleReady waits for a module to be ready
func waitForModuleReady(cluster *KCluster, module ModuleConfig) error {
	// Map module names to their deployment/daemonset names and namespaces
	moduleChecks := map[string]struct {
		namespace  string
		deployment string
		daemonset  string
		timeout    int
	}{
		"snapshot-controller": {
			namespace:  SnapshotControllerModuleNamespace,
			deployment: SnapshotControllerDeploymentName,
			timeout:    int(ModuleDeployTimeout.Seconds()),
		},
		"sds-local-volume": {
			namespace:  SDSLocalVolumeModuleNamespace,
			deployment: SDSLocalVolumeCSIControllerDeploymentName,
			daemonset:  SDSLocalVolumeCSINodeDaemonSetName,
			timeout:    int(ModuleDeployTimeout.Seconds()),
		},
		"sds-node-configurator": {
			namespace: SDSNodeConfiguratorModuleNamespace,
			daemonset: SDSNodeConfiguratorDaemonSetName,
			timeout:   int(ModuleDeployTimeout.Seconds()),
		},
		"sds-replicated-volume": {
			namespace:  SDSReplicatedVolumeModuleNamespace,
			deployment: SDSReplicatedVolumeControllerDeploymentName,
			timeout:    int(ModuleDeployTimeout.Seconds()),
		},
	}

	check, exists := moduleChecks[module.Name]
	if !exists {
		// Module doesn't have specific readiness checks, just wait a bit
		time.Sleep(10 * time.Second)
		return nil
	}

	// Check deployment if specified
	if check.deployment != "" {
		if err := cluster.WaitUntilDeploymentReady(check.namespace, check.deployment, check.timeout); err != nil {
			return fmt.Errorf("deployment %s in namespace %s is not ready: %w", check.deployment, check.namespace, err)
		}
	}

	// Check daemonset if specified
	if check.daemonset != "" {
		if err := cluster.WaitUntilDaemonSetReady(check.namespace, check.daemonset, check.timeout); err != nil {
			return fmt.Errorf("daemonset %s in namespace %s is not ready: %w", check.daemonset, check.namespace, err)
		}
	}

	return nil
}

// UnmarshalYAMLConfig reads, parses, validates, and prepares the cluster definition from YAML
// yamlPath is the path to the YAML configuration file
// Returns the cluster definition, hypervisor cluster (if VM), and any error
func UnmarshalYAMLConfig(yamlPath string) (ClusterDefinition, *KCluster, error) {
	var definition ClusterDefinition
	var hvCluster *KCluster

	// Read YAML file
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return definition, nil, fmt.Errorf("failed to read cluster definition YAML: %w", err)
	}

	// Parse YAML directly into ClusterDefinition
	// Since enum types are string-based, YAML will unmarshal them as strings
	// We need to validate and convert them to the proper enum constants
	if err := yaml.Unmarshal(yamlData, &definition); err != nil {
		return definition, nil, fmt.Errorf("failed to parse cluster definition YAML: %w", err)
	}

	// Validate and convert string values to proper enum types
	// Also validates that all nodes are the same type and required fields are present
	if err := validateAndConvertClusterDefinition(&definition); err != nil {
		return definition, nil, fmt.Errorf("failed to validate cluster definition: %w", err)
	}

	// Resolve SSH keys for all nodes (handle paths, values, or defaults)
	if err := resolveSSHKeys(&definition); err != nil {
		return definition, nil, fmt.Errorf("failed to resolve SSH keys: %w", err)
	}

	// Determine if cluster is VM or bare-metal (all nodes should be the same type after validation)
	isVM := false
	if len(definition.Masters) > 0 {
		isVM = definition.Masters[0].HostType == HostTypeVM
	} else if len(definition.Workers) > 0 {
		isVM = definition.Workers[0].HostType == HostTypeVM
	}

	Infof("Cluster type: %s", map[bool]string{true: "VM", false: "Bare-metal"}[isVM])

	// Set up hypervisor cluster if needed (for VMs)
	if isVM {
		if HypervisorKubeConfig == "" {
			return definition, nil, fmt.Errorf("hypervisor kubeconfig is required for VM clusters but not provided")
		}
		HvSshClient = GetSshClient(HvSshUser, HvHost+":22", HvSshKey)
		go HvSshClient.NewTunnel("127.0.0.1:"+HvK8sPort, "127.0.0.1:"+HvK8sPort)

		var err error
		hvCluster, err = InitKCluster(HypervisorKubeConfig, "")
		if err != nil {
			return definition, nil, fmt.Errorf("failed to initialize hypervisor cluster: %w", err)
		}
	}

	return definition, hvCluster, nil
}

// validateAndConvertClusterDefinition validates and converts string values to proper enum types
// Also ensures all nodes are the same type (all VM or all bare-metal)
func validateAndConvertClusterDefinition(definition *ClusterDefinition) error {
	// Collect all nodes
	allNodes := []*ClusterNode{}
	for i := range definition.Masters {
		allNodes = append(allNodes, &definition.Masters[i])
	}
	for i := range definition.Workers {
		allNodes = append(allNodes, &definition.Workers[i])
	}
	if definition.Setup != nil {
		allNodes = append(allNodes, definition.Setup)
	}

	if len(allNodes) == 0 {
		return fmt.Errorf("cluster definition must contain at least one node")
	}

	// Convert enums and validate each node
	var firstHostType HostType
	for i, node := range allNodes {
		if err := convertNodeEnums(node); err != nil {
			return fmt.Errorf("node %d (%s): %w", i, node.Hostname, err)
		}

		// Check that all nodes are the same type
		if i == 0 {
			firstHostType = node.HostType
		} else if node.HostType != firstHostType {
			return fmt.Errorf("all nodes must be of the same type (VM or bare-metal), but found mixed types: first node is %s, node %d (%s) is %s",
				firstHostType, i, node.Hostname, node.HostType)
		}

		// Validate required fields based on node type
		if err := validateNodeFields(node); err != nil {
			return fmt.Errorf("node %s: %w", node.Hostname, err)
		}
	}

	return nil
}

// validateNodeFields validates required fields based on node type
func validateNodeFields(node *ClusterNode) error {
	// Common required fields
	if node.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if node.Role == "" {
		return fmt.Errorf("role is required")
	}
	if node.Auth.Method == "" {
		return fmt.Errorf("auth.method is required")
	}
	if node.Auth.User == "" {
		return fmt.Errorf("auth.user is required")
	}

	// Type-specific validation
	switch node.HostType {
	case HostTypeVM:
		// VM required fields
		if string(node.OSType) == "" {
			return fmt.Errorf("osType is required for VM nodes")
		}
		if node.CPU <= 0 {
			return fmt.Errorf("cpu is required and must be > 0 for VM nodes")
		}
		if node.RAM <= 0 {
			return fmt.Errorf("ram is required and must be > 0 for VM nodes")
		}
		if node.DiskSize <= 0 {
			return fmt.Errorf("diskSize is required and must be > 0 for VM nodes")
		}
		// ipAddress is optional for VMs (will be set by hypervisor)
	case HostTypeBareMetal:
		// Bare-metal required fields
		if node.IPAddress == "" {
			return fmt.Errorf("ipAddress is required for bare-metal nodes")
		}
		// osType, cpu, ram, diskSize are optional for bare-metal
	default:
		return fmt.Errorf("invalid hostType: %s", node.HostType)
	}

	// Auth method validation
	if node.Auth.Method == AuthMethodSSHPass && node.Auth.Password == "" {
		return fmt.Errorf("auth.password is required when auth.method is ssh-password")
	}

	return nil
}

// convertNodeEnums converts string values to proper enum types for a node
func convertNodeEnums(node *ClusterNode) error {
	// Convert OSType
	osTypeStr := string(node.OSType)
	switch osTypeStr {
	case "Ubuntu_22":
		node.OSType = OSTypeUbuntu22
	case "Ubuntu_24":
		node.OSType = OSTypeUbuntu24
	case "Debian_11":
		node.OSType = OSTypeDebian11
	case "Astra_173":
		node.OSType = OSTypeAstra173
	case "Astra_175":
		node.OSType = OSTypeAstra175
	case "Astra_181":
		node.OSType = OSTypeAstra181
	case "RedOS_7_3":
		node.OSType = OSTypeRedOS73
	case "RedOS_8":
		node.OSType = OSTypeRedOS8
	case "Alt_10":
		node.OSType = OSTypeAlt10
	default:
		return fmt.Errorf("unsupported OS type: %s", osTypeStr)
	}

	// Convert HostType
	hostTypeStr := string(node.HostType)
	switch hostTypeStr {
	case "vm":
		node.HostType = HostTypeVM
	case "bare-metal":
		node.HostType = HostTypeBareMetal
	default:
		return fmt.Errorf("unsupported host type: %s", hostTypeStr)
	}

	// Convert Role
	roleStr := string(node.Role)
	switch roleStr {
	case "master":
		node.Role = ClusterRoleMaster
	case "worker":
		node.Role = ClusterRoleWorker
	case "setup":
		node.Role = ClusterRoleSetup
	default:
		return fmt.Errorf("unsupported role: %s", roleStr)
	}

	// Convert Auth Method
	methodStr := string(node.Auth.Method)
	switch methodStr {
	case "ssh-key":
		node.Auth.Method = AuthMethodSSHKey
	case "ssh-password":
		node.Auth.Method = AuthMethodSSHPass
	default:
		return fmt.Errorf("unsupported auth method: %s", methodStr)
	}

	return nil
}

// resolveSSHKeys resolves SSH key paths/values for all nodes
// For VM clusters: keys are generated in KubePath (handled separately in ensureVMReady)
// For bare-metal clusters or setup nodes: uses public key resolution logic
func resolveSSHKeys(definition *ClusterDefinition) error {
	allNodes := []*ClusterNode{}
	for i := range definition.Masters {
		allNodes = append(allNodes, &definition.Masters[i])
	}
	for i := range definition.Workers {
		allNodes = append(allNodes, &definition.Workers[i])
	}
	if definition.Setup != nil {
		allNodes = append(allNodes, definition.Setup)
	}

	// Determine cluster type
	isVM := false
	if len(definition.Masters) > 0 {
		isVM = definition.Masters[0].HostType == HostTypeVM
	} else if len(definition.Workers) > 0 {
		isVM = definition.Workers[0].HostType == HostTypeVM
	}

	for _, node := range allNodes {
		if node.Auth.Method != AuthMethodSSHKey {
			continue // Only resolve SSH keys for ssh-key auth method
		}

		// For VM clusters, skip resolution - keys will be generated in KubePath during VM creation
		// For bare-metal clusters or setup nodes, resolve public key to find private key
		if !isVM || node.Role == ClusterRoleSetup {
			if err := resolveNodeSSHKey(node); err != nil {
				return fmt.Errorf("node %s: %w", node.Hostname, err)
			}
		}
	}

	return nil
}

// resolveNodeSSHKey resolves SSH keys for a single node
// sshKey in YAML is a public key (for VM creation), but we need the private key for SSH authentication
// If sshKey is empty, uses default ~/.ssh/id_rsa.pub (public) and ~/.ssh/id_rsa (private)
// If sshKey is a public key value (starts with "ssh-rsa", "ssh-ed25519", etc.), finds corresponding private key
// If sshKey is a public key file path (ends with .pub), finds corresponding private key (same name without .pub)
// The public key is kept in node.Auth.SSHKey for VM creation, private key path is stored in node.Auth.privateKeyPath
func resolveNodeSSHKey(node *ClusterNode) error {
	sshKey := node.Auth.SSHKey

	// If empty, use default ~/.ssh/id_rsa.pub (public) and ~/.ssh/id_rsa (private)
	if sshKey == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		defaultPubKey := filepath.Join(homeDir, ".ssh", "id_rsa.pub")
		defaultPrivKey := filepath.Join(homeDir, ".ssh", "id_rsa")

		// Validate private key exists
		if _, err := os.Stat(defaultPrivKey); os.IsNotExist(err) {
			return fmt.Errorf("default SSH private key does not exist: %s", defaultPrivKey)
		}

		node.Auth.SSHKey = defaultPubKey          // Store public key path for VM creation
		node.Auth.privateKeyPath = defaultPrivKey // Store private key path for SSH auth
		Debugf("Using default SSH keys - public: %s, private: %s", defaultPubKey, defaultPrivKey)
		return nil
	}

	// Check if it's a public key value (starts with "ssh-rsa", "ssh-ed25519", "ecdsa-sha2", etc.)
	if strings.HasPrefix(sshKey, "ssh-rsa ") || strings.HasPrefix(sshKey, "ssh-ed25519 ") ||
		strings.HasPrefix(sshKey, "ecdsa-sha2-") || strings.HasPrefix(sshKey, "ssh-dss ") {
		// It's a public key value - need to find corresponding private key
		// Try common locations: ~/.ssh/id_rsa, ~/.ssh/id_ed25519, etc.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		// Try common private key locations
		commonKeys := []string{"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa"}
		var foundPrivKey string
		for _, keyName := range commonKeys {
			privKeyPath := filepath.Join(homeDir, ".ssh", keyName)
			if _, err := os.Stat(privKeyPath); err == nil {
				foundPrivKey = privKeyPath
				break
			}
		}

		if foundPrivKey == "" {
			return fmt.Errorf("public key provided but corresponding private key not found in ~/.ssh/ (tried: %v)", commonKeys)
		}

		node.Auth.SSHKey = sshKey               // Keep public key value for VM creation
		node.Auth.privateKeyPath = foundPrivKey // Store private key path for SSH auth
		Debugf("Using public key value, found private key: %s", foundPrivKey)
		return nil
	}

	// It's a path - expand ~ and check if it's a public key file (.pub)
	expandedPath := sshKey
	if strings.HasPrefix(sshKey, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		expandedPath = filepath.Join(homeDir, sshKey[2:])
	}

	// Check if it's a public key file (ends with .pub)
	if strings.HasSuffix(expandedPath, ".pub") {
		// It's a public key file - find corresponding private key (same name without .pub)
		privKeyPath := strings.TrimSuffix(expandedPath, ".pub")

		// Validate both files exist
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return fmt.Errorf("SSH public key file does not exist: %s", expandedPath)
		}
		if _, err := os.Stat(privKeyPath); os.IsNotExist(err) {
			return fmt.Errorf("SSH private key file does not exist: %s (corresponding to public key %s)", privKeyPath, expandedPath)
		}

		node.Auth.SSHKey = expandedPath        // Store public key path for VM creation
		node.Auth.privateKeyPath = privKeyPath // Store private key path for SSH auth
		Debugf("Using SSH keys - public: %s, private: %s", expandedPath, privKeyPath)
		return nil
	}

	// Path doesn't end with .pub - assume it's a public key file path anyway and try to find private key
	// Validate public key file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH public key file does not exist: %s (note: sshKey should be a public key, not private key)", expandedPath)
	}

	// Try to find corresponding private key (same name without extension, or try common names)
	privKeyPath := expandedPath
	// Try removing common extensions
	if strings.HasSuffix(expandedPath, ".pub") {
		privKeyPath = strings.TrimSuffix(expandedPath, ".pub")
	} else {
		// Try common private key names in the same directory
		dir := filepath.Dir(expandedPath)
		baseName := filepath.Base(expandedPath)
		commonPrivNames := []string{
			filepath.Join(dir, strings.TrimSuffix(baseName, filepath.Ext(baseName))),
			filepath.Join(dir, "id_rsa"),
			filepath.Join(dir, "id_ed25519"),
		}
		for _, privName := range commonPrivNames {
			if _, err := os.Stat(privName); err == nil {
				privKeyPath = privName
				break
			}
		}
	}

	if _, err := os.Stat(privKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH private key file does not exist: %s (corresponding to public key %s). Please ensure the private key exists", privKeyPath, expandedPath)
	}

	node.Auth.SSHKey = expandedPath        // Store public key path for VM creation
	node.Auth.privateKeyPath = privKeyPath // Store private key path for SSH auth
	Debugf("Using SSH keys - public: %s, private: %s", expandedPath, privKeyPath)
	return nil
}
