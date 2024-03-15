package test

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sds-lvm-e2e/funcs"
	"testing"
	"time"
)

const (
	immediateStorageClassName            = "e2e-test-immediate"
	waitForFirstConsumerStorageClassName = "ssd-whole-cluster"

	count      = 600
	volumeSize = "50Gi"
)

func BenchmarkRunFullCycleImmediateStorageClassSingleThread(b *testing.B) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		b.Error(err)
	}

	b.StartTimer()
	b.Log("------------  pvc creating ------------- ")
	pvcNames := make([]string, 0, count)
	for i := 0; i < count; i++ {
		pvcName := fmt.Sprintf("test-pvc-%d", i)
		_, err := funcs.CreatePVC(
			ctx,
			cl,
			pvcName,
			immediateStorageClassName,
			volumeSize,
			false)
		if err != nil {
			b.Error(err)
			continue
		}

		pvcNames = append(pvcNames, pvcName)
	}

	bounded := 0
	for bounded < len(pvcNames) {
		time.Sleep(100 * time.Millisecond)
		for _, pvcName := range pvcNames {
			status, err := funcs.GetPVCStatusPhase(ctx, cl, pvcName)
			if err != nil {
				if kerrors.IsNotFound(err) {
					b.Logf("waiting for PVC %s to be created", pvcName)
				} else {
					b.Error(err)
				}

				continue
			}

			if status == v1.ClaimBound {
				b.Logf("PVC %s was successfully bounded", pvcName)
				bounded++
			}
		}
	}
	b.Logf("[TIME] All PVC were bounded for %s", b.Elapsed().String())

	b.Log("------------  pod creating ------------- ")
	command := []string{"/bin/bash"}
	args := []string{"-c", "df -T | grep '/usr/share/test-data' | grep 'ext4'"}

	podNames := make(map[string]struct{}, len(pvcNames))
	for _, pvcName := range pvcNames {
		podName := fmt.Sprintf("test-pvc-%s-pod", pvcName)
		_, err := funcs.CreatePod(ctx, cl, podName, pvcName, false, command, args)
		if err != nil {
			b.Error(err)
			continue
		}

		podNames[podName] = struct{}{}
		b.Log(fmt.Sprintf("pod=%s created", podName))
	}

	succeeded := 0
	for succeeded < len(podNames) {
		time.Sleep(100 * time.Millisecond)
		for podName := range podNames {
			phase, err := funcs.GetPodStatusPhase(ctx, cl, podName)
			if err != nil {
				b.Error(err)
				continue
			}

			if phase == v1.PodRunning {
				b.Logf("Pod %s in a running state", podName)
				succeeded++
			}
		}
	}
	b.Logf("[TIME] All pod got running for %s", b.Elapsed().String())
	b.StopTimer()

	b.Log("------------  pod deleting ------------- ")
	for podName := range podNames {
		err = funcs.DeletePod(ctx, cl, podName)
		if err != nil {
			b.Error(err)
		}
	}

	deleted := 0
	for deleted < len(podNames) {
		time.Sleep(100 * time.Millisecond)
		for podName := range podNames {
			del, err := funcs.IsPodDeleted(ctx, cl, podName)
			if err != nil && !del {
				b.Log(err.Error())
			}

			if del {
				b.Logf("Pod %s was successfully deleted", podName)
				deleted++
			}
		}
	}

	b.Log("------------  pvc deleting ------------- ")
	for _, pvcName := range pvcNames {
		err = funcs.DeletePVC(ctx, cl, pvcName)
		if err != nil {
			b.Error(err)
		}
	}

	deleted = 0
	for deleted < len(pvcNames) {
		time.Sleep(100 * time.Millisecond)
		for _, pvcName := range pvcNames {
			del, err := funcs.IsPVCDeleted(ctx, cl, pvcName)
			if err != nil && !del {
				b.Error(err)
			}

			if del {
				b.Logf("PVC %s was successfully deleted", pvcName)
				deleted++
			}
		}
	}
}

func BenchmarkRunFullCycleWFFCStorageClassSingleThread(b *testing.B) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		b.Error(err)
	}

	b.StartTimer()
	b.Log("------------  pvc creating ------------- ")
	pvcNames := make(map[string]bool, count)
	for i := 0; i < count; i++ {
		pvcName := fmt.Sprintf("test-pvc-%d", i)
		_, err := funcs.CreatePVC(
			ctx,
			cl,
			pvcName,
			waitForFirstConsumerStorageClassName,
			volumeSize,
			false)
		if err != nil {
			b.Error(err)
			continue
		}

		pvcNames[pvcName] = false
	}
	b.Logf("[TIME] All PVC were created for %s", b.Elapsed().String())

	b.Log("------------  pod creating ------------- ")
	command := []string{"/bin/bash"}
	args := []string{"-c", "df -T | grep '/usr/share/test-data' | grep 'ext4'"}

	podNames := make(map[string]bool, len(pvcNames))
	for pvcName := range pvcNames {
		podName := fmt.Sprintf("test-pvc-%s-pod", pvcName)
		_, err := funcs.CreatePod(ctx, cl, podName, pvcName, false, command, args)
		if err != nil {
			b.Error(err)
			continue
		}

		podNames[podName] = false
		b.Log(fmt.Sprintf("pod=%s created", podName))
	}
	b.Logf("[TIME] All Pods were created for %s", b.Elapsed().String())

	b.Log("------------  pvc bounding ------------- ")
	bounded := 0
	for bounded < len(pvcNames) {
		//time.Sleep(100 * time.Millisecond)
		pvcs, err := funcs.GetPVCs(ctx, cl)
		if err != nil {
			b.Error(err)
			continue
		}

		for pvcName, bound := range pvcNames {
			if pvc, found := pvcs[pvcName]; found {
				if pvc.Status.Phase == v1.ClaimBound && !bound {
					b.Logf("PVC %s was successfully bounded", pvcName)
					pvcNames[pvcName] = true
					bounded++
				}
			}
		}
	}
	b.Logf("[TIME] All PVC were bounded for %s", b.Elapsed().String())

	b.Log("------------  pods running ------------- ")
	succeeded := 0
	for succeeded < len(podNames) {
		//time.Sleep(100 * time.Millisecond)
		pods, err := funcs.GetPods(ctx, cl)
		if err != nil {
			b.Error(err)
			continue
		}

		for podName, success := range podNames {
			if pod, found := pods[podName]; found {
				if pod.Status.Phase == v1.PodSucceeded && !success {
					b.Logf("Pod %s in a compeleted state", podName)
					podNames[podName] = true
					succeeded++
				}
			}
		}
	}
	b.Logf("[TIME] All pod got compeleted for %s", b.Elapsed().String())
	b.StopTimer()

	//b.Log("------------  pod deleting ------------- ")
	//for podName := range podNames {
	//	err = funcs.DeletePod(ctx, cl, podName)
	//	if err != nil {
	//		b.Error(err)
	//	}
	//}

	//deleted := 0
	//for deleted < len(podNames) {
	//	time.Sleep(100 * time.Millisecond)
	//	for podName := range podNames {
	//		del, err := funcs.IsPodDeleted(ctx, cl, podName)
	//		if err != nil && !del {
	//			b.Log(err.Error())
	//		}
	//
	//		if del {
	//			b.Logf("Pod %s was successfully deleted", podName)
	//			deleted++
	//		}
	//	}
	//}
	//
	//b.Log("------------  pvc deleting ------------- ")
	//for pvcName := range pvcNames {
	//	err = funcs.DeletePVC(ctx, cl, pvcName)
	//	if err != nil {
	//		b.Error(err)
	//	}
	//}
	//
	//deleted = 0
	//for deleted < len(pvcNames) {
	//	time.Sleep(100 * time.Millisecond)
	//	for pvcName := range pvcNames {
	//		del, err := funcs.IsPVCDeleted(ctx, cl, pvcName)
	//		if err != nil && !del {
	//			b.Error(err)
	//		}
	//
	//		if del {
	//			b.Logf("PVC %s was successfully deleted", pvcName)
	//			deleted++
	//		}
	//	}
	//}
}
