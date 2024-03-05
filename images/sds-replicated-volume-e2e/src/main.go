/*
Copyright 2023 Flant JSC

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

package main

import (
	"sds-replicated-volume-e2e/tests"
)

func main() {
	_, err := test.NewKubeClient()
	if err != nil {
		panic(err)
	}

	//klog.Info("Creating Pools")
	//funcs.CreatePools(*dynamicClient)
	//
	//klog.Info("Testing Logs STS")
	//funcs.CreateLogSts(*clientset)
}
