package integration

import (
	"testing"
	"os"
	"flag"
)

/*
var fakepubsubNodePort = flag.Int("fakepubsub-node-port", 30303, "The port to use for connecting sub tests with the fakepubsub service (for configuring PUBSUB_EMULATOR_HOST)")
var (
	clusterNameFlag = flag.String("cluster", "dev", "The context of cluster to use for test")
	vmOS = flag.String("virtos", "", "Deploy virtual machine with specified OS")
)
*/

func TestMain(m *testing.M) {
    flag.Parse()

	//TODO kubectl delete ns test1

	//TODO if bare metal ...
	// configs in env
	// InitClusterCreate()

	//TODO if dev stand

	//TODO if lockal (docker)

	//TODO else default

	//TODO func VirtClusterCreate() {}


    os.Exit(m.Run())
}
