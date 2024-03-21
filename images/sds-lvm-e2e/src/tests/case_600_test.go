package test

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"sds-lvm-e2e/funcs"
	"testing"
)

const (
	immediateStorageClassName            = "ssd-immediate-thick-perf-test"
	waitForFirstConsumerStorageClassName = "ssd-wffc-thick-perf-test"

	count      = 600
	volumeSize = "1Gi"
)

func BenchmarkRunFullCycleImmediateStorageClassSingleThread(b *testing.B) {
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
			immediateStorageClassName,
			volumeSize,
			false)
		if err != nil {
			b.Error(err)
			continue
		}

		pvcNames[pvcName] = false
	}
	b.Logf("[TIME] All PVC were created for %s", b.Elapsed().String())

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
	b.Logf("[TIME] All pod got compeleted for %fs", b.Elapsed().Seconds())
	b.StopTimer()

	//b.Log("------------  pod deleting ------------- ")
	//for podName := range podNames {
	//	err = funcs.DeletePod(ctx, cl, podName)
	//	if err != nil {
	//		b.Error(err)
	//	}
	//}
	//
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
	//for _, pvcName := range pvcNames {
	//	err = funcs.DeletePVC(ctx, cl, pvcName)
	//	if err != nil {
	//		b.Error(err)
	//	}
	//}
	//
	//deleted = 0
	//for deleted < len(pvcNames) {
	//	time.Sleep(100 * time.Millisecond)
	//	for _, pvcName := range pvcNames {
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
	b.Logf("[TIME] All pod got compeleted for %fs", b.Elapsed().Seconds())
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
