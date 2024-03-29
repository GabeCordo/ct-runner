package clusters

import (
	"github.com/GabeCordo/processor-framework/processor"
	"github.com/GabeCordo/processor-framework/processor/components/cluster"
)

func LinkCommon(processor *processor.Processor) {

	mod := processor.Module("common")
	mod.Version = 1.0
	mod.Mounted = true

	v := VectorCluster{}
	ccfg := &cluster.Config{
		Identifier:                  "vec",
		OnLoad:                      cluster.CompleteAndPush,
		OnCrash:                     cluster.DoNothing,
		StartWithNTransformClusters: 1,
		StartWithNLoadClusters:      1,
		ETChannelThreshold:          1,
		ETChannelGrowthFactor:       2,
		TLChannelThreshold:          1,
		TLChannelGrowthFactor:       2,
	}
	mod.AddCluster("vec", string(cluster.Batch), v, ccfg)

	h := HelloCluster{}
	hcfg := &cluster.Config{
		Identifier:                  "hello",
		OnLoad:                      cluster.CompleteAndPush,
		OnCrash:                     cluster.DoNothing,
		StartWithNTransformClusters: 1,
		StartWithNLoadClusters:      1,
		ETChannelThreshold:          1,
		ETChannelGrowthFactor:       2,
		TLChannelThreshold:          1,
		TLChannelGrowthFactor:       2,
	}
	mod.AddCluster("hello", string(cluster.Batch), h, hcfg)
}
