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

package stress

import "flag"

var (
	testNamespace    = *flag.String("namespace", "sds-replicated-volume-e2e-test", "namespace in which test runs")
	stsCount         = *flag.Int("stsCount", 50, "number of sts instances")
	pvSize           = *flag.String("pvSize", "5Gi", "size of PV in Gi")
	pvResizedSize    = *flag.String("pvResizedSize", "5.1Gi", "size of resized PV in Gi")
	storageClassName = *flag.String("storageClassName", "linstor-r2", "storage class name")
	createRSP        = *flag.Bool("createRSP", true, "create RSP")
	createVM         = *flag.Bool("createVM", true, "create VM")
)
