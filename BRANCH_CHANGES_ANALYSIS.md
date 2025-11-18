# Branch Changes Analysis: create-cluster-fix

## Summary
This analysis examines changes in the `create-cluster-fix` branch and their impact on tests in the `testkit_v2/tests/` folder (excluding `00_healthcheck_test.go`).

## Changes in the Branch

### 1. Modified Files

#### `testkit_v2/util/kube_vm_cluster.go` (Major Changes)
- **Added `ensureNodesReady()` function**: New function that waits for all nodes to be ready before checking module readiness
  - Uses `NodesReadyTimeout = 600` seconds (10 minutes)
  - Checks node Ready condition before proceeding
  
- **Completely rewrote `ensureClusterReady()` function**:
  - **Before**: Only checked `sds-node-configurator` daemonset readiness
  - **After**: Now checks modules in a specific dependency order:
    1. `snapshot-controller` module (deployment readiness)
    2. `sds-local-volume` module (deployment + daemonset readiness)
    3. `sds-node-configurator` module (daemonset readiness)
  - Added helper functions: `checkDeploymentReady()` and `checkDaemonSetReady()`
  - More thorough validation of daemonset pods (checks all pods are running)

- **Added `ensureNodesReady()` call** in `ClusterCreate()` before `ensureClusterReady()`

#### `testkit_v2/util/kube_deckhouse_modules.go`
- Added constants for `snapshot-controller` and `sds-local-volume` modules
- Increased `ModuleReadyTimeout` from 600 to 720 seconds (12 minutes)

#### `testkit_v2/data/resources.yml.tpl`
- Added `snapshot-controller` module configuration:
  - `ModulePullOverride` for snapshot-controller
  - `ModuleConfig` to enable snapshot-controller

#### Other Files
- `.gitignore`: Added generated config files (no test impact)
- `testkit_v2/runme.sh`: New script (no test impact)

## Impact on Tests

### Affected Tests

**ALL tests that use `util.EnsureCluster("", "")` with `HypervisorKubeConfig` set will be affected** because:

1. `EnsureCluster()` calls `ClusterCreate()` when `HypervisorKubeConfig != ""`
2. `ClusterCreate()` calls the modified `ensureClusterReady()` function
3. The new `ensureClusterReady()` has stricter requirements and different behavior

### Specific Test Files Affected

#### ✅ `01_sds_nc_test.go` - **AFFECTED**
- **Impact**: HIGH
- Uses `util.EnsureCluster("", "")` which triggers cluster creation
- Also uses `util.EnsureCluster(util.HypervisorKubeConfig, "")` for hypervisor operations
- **Changes affect**: Cluster initialization will now wait for snapshot-controller and sds-local-volume modules before proceeding
- **Potential issues**: 
  - Tests may take longer to start (additional module checks)
  - Tests may fail if snapshot-controller or sds-local-volume modules are not properly configured
  - Node readiness check added before module checks

#### ✅ `03_sds_lv_test.go` - **AFFECTED**
- **Impact**: MEDIUM
- Uses `util.EnsureCluster("", "")` for cluster operations
- **Changes affect**: Cluster initialization with stricter module readiness checks
- **Potential issues**: 
  - Tests may fail if required modules are not ready
  - Longer initialization time

#### ✅ `05_sds_node_configurator_test.go` - **AFFECTED**
- **Impact**: HIGH
- Uses `util.EnsureCluster("", "")` extensively
- Also uses `util.EnsureCluster(util.HypervisorKubeConfig, "")` for hypervisor operations
- **Changes affect**: 
  - All LVM thick/thin tests depend on cluster being ready
  - Module readiness checks are now more comprehensive
- **Potential issues**:
  - Tests may fail if snapshot-controller is not available
  - Tests may fail if sds-local-volume module is not properly configured
  - Node readiness validation added

#### ✅ `99_finalizer_test.go` - **AFFECTED**
- **Impact**: LOW
- Uses `util.EnsureCluster("", "")`
- **Changes affect**: Cluster initialization
- **Potential issues**: Minimal - this is a cleanup test

#### ✅ `data-exporter/base_test.go` - **AFFECTED**
- **Impact**: MEDIUM
- Uses `util.EnsureCluster("", "")`
- **Changes affect**: Cluster initialization before PVC creation
- **Potential issues**: 
  - PVC creation may fail if storage modules are not ready
  - Tests may take longer to initialize

#### ✅ `tools.go` - **AFFECTED**
- **Impact**: MEDIUM
- Contains helper functions used by other tests
- Uses `util.EnsureCluster(util.HypervisorKubeConfig, "")` for hypervisor operations
- **Changes affect**: Helper functions that create/cleanup resources

## Key Behavioral Changes

### 1. Module Readiness Order
**Before**: Only checked `sds-node-configurator` daemonset
```go
// Old behavior - simple check
dsNodeConfigurator, err := cluster.GetDaemonSet("d8-sds-node-configurator", "sds-node-configurator")
if int(dsNodeConfigurator.Status.NumberReady) < len(VmCluster) {
    return fmt.Errorf("sds-node-configurator ready: %d of %d", ...)
}
```

**After**: Checks modules in dependency order with comprehensive validation
```go
// New behavior - sequential checks
1. Check snapshot-controller deployment
2. Check sds-local-volume deployment + daemonset
3. Check sds-node-configurator daemonset (with pod validation)
```

### 2. Node Readiness Check
**New**: Added `ensureNodesReady()` check before module readiness
- Waits up to 10 minutes for all nodes to be in Ready state
- Validates node conditions before proceeding

### 3. Timeout Changes
- `ModuleReadyTimeout`: 600s → 720s (20% increase)
- Added `NodesReadyTimeout`: 600s (new)

### 4. Daemonset Validation
**Enhanced**: Now validates:
- Desired == Current == Ready counts
- All pods are in Running phase
- Pod count matches expected

## Potential Issues & Recommendations

### ⚠️ Critical Issues

1. **Missing Module Dependencies**
   - If `snapshot-controller` module is not properly configured in the cluster, ALL tests will fail
   - If `sds-local-volume` module is not ready, tests will fail
   - **Action**: Ensure `resources.yml.tpl` changes are applied to test clusters

2. **Timeout Increases**
   - Tests may take significantly longer to initialize
   - Total timeout could be: 10min (nodes) + 12min (modules) = 22 minutes worst case
   - **Action**: Consider if test timeouts need adjustment

3. **Stricter Validation**
   - Tests that previously passed with partially ready modules may now fail
   - **Action**: Review test expectations and module configurations

### ✅ Positive Changes

1. **Better Dependency Management**: Ensures modules are ready in correct order
2. **More Robust Validation**: Checks all pods are running, not just daemonset status
3. **Node Readiness**: Ensures nodes are ready before checking modules

## Testing Recommendations

1. **Run all affected tests** to verify they still pass with new cluster readiness checks
2. **Monitor test execution time** - ensure timeouts are sufficient
3. **Verify module configurations** - ensure snapshot-controller and sds-local-volume are properly configured
4. **Check for flaky tests** - stricter checks may reveal timing issues

## Files to Review

- `testkit_v2/tests/01_sds_nc_test.go` - High impact
- `testkit_v2/tests/03_sds_lv_test.go` - Medium impact  
- `testkit_v2/tests/05_sds_node_configurator_test.go` - High impact
- `testkit_v2/tests/99_finalizer_test.go` - Low impact
- `testkit_v2/tests/data-exporter/base_test.go` - Medium impact
- `testkit_v2/tests/tools.go` - Medium impact

## Conclusion

**All tests in the `tests/` folder (except `00_healthcheck_test.go`) are potentially affected** by these changes because they all use `EnsureCluster()` which triggers `ClusterCreate()` when `HypervisorKubeConfig` is set. The changes make cluster initialization more robust but also more strict, which could cause previously passing tests to fail if module dependencies are not properly configured.

