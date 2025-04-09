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

package test

import (
	"sds-lvm-e2e/funcs"
	"strconv"
	"testing"
	"time"
)

// worker-1
func TestCreatePV(t *testing.T) {
	command, err := funcs.CreatePV("/dev/vdd") // blockDevice
	if err != nil {
		t.Log(command)
		t.Error(err)
	}
}

func TestCreateVGLocal(t *testing.T) {
	command, err := funcs.CreateVGLocal("new", []string{"/dev/vdd"})
	if err != nil {
		t.Log(command)
		t.Error(err)
	}
}

func TestCreateThinPool(t *testing.T) {
	command, err := funcs.CreateThinPool("newpool", "35G", "new")
	if err != nil {
		t.Log(command)
		t.Error(err)
	}
}

func BenchmarkCreateThinLVSerial(b *testing.B) {
	b.N = 100
	for i := 0; i < b.N; i++ {
		command, err := funcs.CreateThinLogicalVolume("new", "newpool", strconv.Itoa(i), "1Gi")
		if err != nil {
			b.Log(command)
			b.Error(err)
		}
	}
}

func BenchmarkDeleteThinLVSerial(b *testing.B) {
	b.N = 20
	for i := 0; i < b.N; i++ {
		command, err := funcs.RemoveLV("new", strconv.Itoa(i))
		if err != nil {
			b.Log(command)
			b.Error(err)
		}
	}
}

func BenchmarkCreateThickLVSerial(b *testing.B) {
	b.N = 20
	for i := 1; i <= b.N; i++ {
		command, err := funcs.CreateThickLogicalVolume("new", strconv.Itoa(i), "1G")
		if err != nil {
			b.Log(command)
			b.Error(err)
		}
	}
}

func BenchmarkDeleteThickLVSerial(b *testing.B) {
	b.N = 20
	for i := 1; i <= b.N; i++ {
		command, err := funcs.RemoveLV("new", strconv.Itoa(i))
		if err != nil {
			b.Log(command)
			b.Error(err)
		}
	}
}

func BenchmarkDeleteThinLVParallel(b *testing.B) {
	var i int
	b.SetParallelism(100)
	b.N = 20
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			command, err := funcs.RemoveLV("new", strconv.Itoa(i))
			if err != nil {
				b.Log(command)
				b.Error(err)
			}
		}
	})
}

func BenchmarkCreateThinLVParallel(b *testing.B) {
	var i int
	b.SetParallelism(100)
	b.N = 20
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			command, err := funcs.CreateThinLogicalVolume("new", "newpool", strconv.Itoa(i), "1G")
			if err != nil {
				b.Log(command)
				b.Error(err)
			}
		}
	})
}

func BenchmarkCreateThickLVParallel(b *testing.B) {
	var i int
	b.SetParallelism(100)
	b.N = 20
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			command, err := funcs.CreateThickLogicalVolume("new", strconv.Itoa(i), "1G")
			if err != nil {
				b.Log(command)
				b.Error(err)
			}
			//time.Sleep(100 * time.Millisecond)
		}
	})
}

func BenchmarkDeleteThickLVParallel(b *testing.B) {
	var i int
	//b.SetParallelism(100)
	b.N = 20
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			b.Log("Delete=", i)
			command, err := funcs.RemoveLV("new", strconv.Itoa(i))
			if err != nil {
				b.Log(command)
				b.Error(err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}

func BenchmarkExtendThickLVParallel(b *testing.B) {
	var i int
	b.SetParallelism(100)
	b.N = 20
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			b.Log("Extend " + strconv.Itoa(i))
			commandExtend, err := funcs.ExtendLV("2G", "new", strconv.Itoa(i))
			if err != nil {
				b.Log(commandExtend)
				b.Error(err)
			}
		}
	})
}

func BenchmarkCreateAndExtendLVSerial(b *testing.B) {
	b.N = 20
	for i := 1; i <= b.N; i++ {
		b.Log("Create " + strconv.Itoa(i))
		command, err := funcs.CreateThickLogicalVolume("new", strconv.Itoa(i), "1G")
		if err != nil {
			b.Log(command)
			b.Error(err)
		}
		b.Log("Extend " + strconv.Itoa(i))
		commandExtend, err := funcs.ExtendLV("1.8G", "new", strconv.Itoa(i))
		if err != nil {
			b.Log(commandExtend)
			b.Error(err)
		}
	}
}

func TestRemoveVGLocal(t *testing.T) {
	command, err := funcs.RemoveVG("new")
	if err != nil {
		t.Log(command)
		t.Error(err)
	}
}

func TestRemovePV(t *testing.T) {
	command, err := funcs.RemovePV([]string{"/dev/vdd"})
	if err != nil {
		t.Log(command)
		t.Error(err)
	}
}

func BenchmarkCreatePV(b *testing.B) {
	b.N = 600
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			command, err := funcs.CreatePV("/dev/sdx")
			if err != nil {
				b.Log(command)
				b.Error(err)
			}
		}
	})
}
