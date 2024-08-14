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
