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
	"context"
	"time"

	"github.com/deckhouse/sds-e2e/util/utiltype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WaitIterationCountDataExport = 10
	WaitIterationCountPVC        = 15
)

func (clr *KCluster) CreateDataExport(exportType, exportName, dataExportName, TTL string, isPublish bool) (*utiltype.DataExport, error) {
	dataExport := &utiltype.DataExport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DataExport",
			APIVersion: "deckhouse.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dataExportName,
			Namespace: TestNS,
		},
		Spec: utiltype.DataexportSpec{
			Ttl:     TTL,
			Publish: isPublish,
			TargetRef: utiltype.TargetRefSpec{
				Kind: exportType,
				Name: exportName,
			},
		},
	}

	err := clr.rtClient.Create(clr.ctx, dataExport)
	if err != nil {
		return nil, err
	}

	return dataExport, nil
}

func (clr *KCluster) CreateDataExportForVirtualDisk(vdName, dataExportName string) (*utiltype.DataExport, error) {
	dataExport := &utiltype.DataExport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DataExport",
			APIVersion: "deckhouse.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dataExportName,
			Namespace: TestNS,
		},
		Spec: utiltype.DataexportSpec{
			Ttl:     "1h",
			Publish: true,
			TargetRef: utiltype.TargetRefSpec{
				Kind: "VirtualDisk",
				Name: vdName,
			},
		},
	}

	err := clr.rtClient.Create(clr.ctx, dataExport)
	if err != nil {
		return nil, err
	}

	return dataExport, nil
}

func (clr *KCluster) GetDataExport(name, namespace string) (*utiltype.DataExport, error) {
	dataExport := &utiltype.DataExport{}
	if err := clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{Name: name, Namespace: namespace}, dataExport); err != nil {
		return nil, err
	}
	return dataExport, nil
}

func (clr *KCluster) WaitDataExportURLReady(name string) (*utiltype.DataExport, error) {
	dataExport := &utiltype.DataExport{}
	for i := 0; i < WaitIterationCountDataExport; i++ {
		Infof("Waiting for the DataExport url to be ready. Attempt %d of %d", i+1, WaitIterationCountDataExport)

		err := clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{
			Name:      name,
			Namespace: TestNS,
		}, dataExport)
		if err != nil {
			Debugf("Failed to get DataExport: %s", err.Error())
			time.Sleep(WaitIterationCountPVC * time.Second)
			continue
		}

		if dataExport.Status.PublicURL != "" && dataExport.Status.Url != "" {
			return dataExport, nil
		}

		Infof("DataExport URL not ready. Trying again...")
		time.Sleep(WaitIterationCountPVC * time.Second)
	}

	return dataExport, nil
}

func (clr *KCluster) DeleteDataExport(name string) error {
	dataExport := &utiltype.DataExport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DataExport",
			APIVersion: "deckhouse.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNS,
		},
	}

	cwt, cancel := context.WithTimeout(clr.ctx, 5*time.Second)
	defer cancel()
	return clr.rtClient.Delete(cwt, dataExport)
}
