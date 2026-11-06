// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package reclaimable

import (
	"testing"

	commonconstants "github.com/kai-scheduler/KAI-scheduler/pkg/common/constants"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/common_info"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/pod_info"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/pod_status"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/podgroup_info"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/podgroup_info/subgroup_info"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/resource_info"
	rs "github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/plugins/proportion/resource_share"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testVectorMap = resource_info.NewResourceVectorMap()

func testPodInfo(uid common_info.PodID, gpus float64, status pod_status.PodStatus) *pod_info.PodInfo {
	return &pod_info.PodInfo{
		UID:            uid,
		GpuRequirement: *resource_info.NewGpuResourceRequirementWithGpus(gpus, 0),
		ResReqVector:   resource_info.NewResourceVectorWithValues(0, 0, gpus, testVectorMap),
		VectorMap:      testVectorMap,
		Status:         status,
	}
}

func testReclaimee(name string, queue common_info.QueueID, pods pod_info.PodsMap) *podgroup_info.PodGroupInfo {
	return &podgroup_info.PodGroupInfo{
		Name:      name,
		Queue:     queue,
		VectorMap: testVectorMap,
		PodSets: map[string]*subgroup_info.PodSet{
			podgroup_info.DefaultSubGroup: subgroup_info.NewPodSet(podgroup_info.DefaultSubGroup, 1, nil).WithPodInfos(pods),
		},
	}
}

type queuesTestData struct {
	parentQueue common_info.QueueID
	deserved    float64
	fairShare   float64
	allocated   float64
}

func TestReclaimable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test reclaimable")
}

var _ = Describe("Can Reclaim Resources", func() {
	Context("Preemptible job", func() {
		tests := []struct {
			name          string
			reclaimerInfo *ReclaimerInfo
			queue         *rs.QueueAttributes
			canReclaim    bool
		}{
			{
				name: "No allocated resources, below fair share",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1000, 1000, 1).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 2,
							Allocated: 0,
						},
						CPU: rs.ResourceShare{
							FairShare: 2000,
							Allocated: 0,
						},
						Memory: rs.ResourceShare{
							FairShare: 2000,
							Allocated: 0,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "Some resources allocated, below fair share",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1000, 1000, 1).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 3,
							Allocated: 1,
						},
						CPU: rs.ResourceShare{
							FairShare: 3000,
							Allocated: 1000,
						},
						Memory: rs.ResourceShare{
							FairShare: 3000,
							Allocated: 1000,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "Some resources at fair share with job, other stay below",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(500, 1000, 0).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 2,
							Allocated: 1,
						},
						CPU: rs.ResourceShare{
							FairShare: 2000,
							Allocated: 1000,
						},
						Memory: rs.ResourceShare{
							FairShare: 2000,
							Allocated: 1000,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "Exactly at fair share with job",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1000, 1000, 1).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 2,
							Allocated: 1,
						},
						CPU: rs.ResourceShare{
							FairShare: 2000,
							Allocated: 1000,
						},
						Memory: rs.ResourceShare{
							FairShare: 2000,
							Allocated: 1000,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "Partially above fair share",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 1,
							Allocated: 1,
						},
						CPU: rs.ResourceShare{
							FairShare: 1500,
							Allocated: 1000,
						},
						Memory: rs.ResourceShare{
							FairShare: 3000,
							Allocated: 1000,
						},
					},
				},
				canReclaim: false,
			},
			{
				name: "Fully above fair share",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 1,
							Allocated: 1,
						},
						CPU: rs.ResourceShare{
							FairShare: 1500,
							Allocated: 1000,
						},
						Memory: rs.ResourceShare{
							FairShare: 1500,
							Allocated: 1000,
						},
					},
				},
				canReclaim: false,
			},
			{
				name: "Queue partially above fair share without job resources",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     true,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							FairShare: 0,
							Allocated: 1,
						},
						CPU: rs.ResourceShare{
							FairShare: 3000,
							Allocated: 2000,
						},
						Memory: rs.ResourceShare{
							FairShare: 3000,
							Allocated: 1000,
						},
					},
				},
				canReclaim: false,
			},
		}

		for _, data := range tests {
			testData := data
			It(testData.name, func() {
				reclaimable := New(1.0)
				queues := map[common_info.QueueID]*rs.QueueAttributes{
					testData.queue.UID: testData.queue,
				}
				result := reclaimable.CanReclaimResources(queues, testData.reclaimerInfo)
				Expect(result).To(Equal(testData.canReclaim))
			})
		}
	})

	Context("Non-Preemptible job", func() {
		tests := []struct {
			name          string
			reclaimerInfo *ReclaimerInfo
			queue         *rs.QueueAttributes
			canReclaim    bool
		}{
			{
				name: "No allocated resources, below quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1000, 1000, 1).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                2,
							FairShare:               2,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
						CPU: rs.ResourceShare{
							Deserved:                2000,
							FairShare:               2000,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
						Memory: rs.ResourceShare{
							Deserved:                2000,
							FairShare:               2000,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "No allocated resources, exactly at quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1000, 1000, 1).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                1,
							FairShare:               2,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
						CPU: rs.ResourceShare{
							Deserved:                1000,
							FairShare:               2000,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
						Memory: rs.ResourceShare{
							Deserved:                1000,
							FairShare:               2000,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "No allocated resources, partially above quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                0,
							FairShare:               2,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
						CPU: rs.ResourceShare{
							Deserved:                1000,
							FairShare:               2000,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
						Memory: rs.ResourceShare{
							Deserved:                500,
							FairShare:               2000,
							Allocated:               0,
							AllocatedNotPreemptible: 0,
						},
					},
				},
				canReclaim: false,
			},
			{
				name: "Some resources allocated, below quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1000, 1000, 1).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                3,
							FairShare:               3,
							Allocated:               1,
							AllocatedNotPreemptible: 1,
						},
						CPU: rs.ResourceShare{
							Deserved:                3000,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
						Memory: rs.ResourceShare{
							Deserved:                3000,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
					},
				},
				canReclaim: true,
			},
			{
				name: "Some preemptible resources allocated, zero quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                0,
							FairShare:               3,
							Allocated:               1,
							AllocatedNotPreemptible: 0,
						},
						CPU: rs.ResourceShare{
							Deserved:                0,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 0,
						},
						Memory: rs.ResourceShare{
							Deserved:                0,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 0,
						},
					},
				},
				canReclaim: false,
			},
			{
				name: "Partially above quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                0,
							FairShare:               3,
							Allocated:               1,
							AllocatedNotPreemptible: 1,
						},
						CPU: rs.ResourceShare{
							Deserved:                1500,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
						Memory: rs.ResourceShare{
							Deserved:                3000,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
					},
				},
				canReclaim: false,
			},
			{
				name: "Fully above quota",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                1,
							FairShare:               3,
							Allocated:               1,
							AllocatedNotPreemptible: 1,
						},
						CPU: rs.ResourceShare{
							Deserved:                1500,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
						Memory: rs.ResourceShare{
							Deserved:                1500,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
					},
				},
				canReclaim: false,
			},
			{
				name: "Queue partially over quota without job resources",
				reclaimerInfo: &ReclaimerInfo{
					Queue:             "queue1",
					RequiredResources: resource_info.NewResource(1, 1000, 1000).ToVector(testVectorMap),
					VectorMap:         testVectorMap,
					IsPreemptable:     false,
				},
				queue: &rs.QueueAttributes{
					UID:         "queue1",
					ParentQueue: "",
					QueueResourceShare: rs.QueueResourceShare{
						GPU: rs.ResourceShare{
							Deserved:                0,
							FairShare:               3,
							Allocated:               1,
							AllocatedNotPreemptible: 1,
						},
						CPU: rs.ResourceShare{
							Deserved:                1500,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
						Memory: rs.ResourceShare{
							Deserved:                3000,
							FairShare:               3000,
							Allocated:               1000,
							AllocatedNotPreemptible: 1000,
						},
					},
				},
				canReclaim: false,
			},
		}

		for _, data := range tests {
			testData := data
			It(testData.name, func() {
				reclaimable := New(1.0)
				queues := map[common_info.QueueID]*rs.QueueAttributes{
					testData.queue.UID: testData.queue,
				}
				result := reclaimable.CanReclaimResources(queues, testData.reclaimerInfo)
				Expect(result).To(Equal(testData.canReclaim))
			})
		}
	})
})

var _ = Describe("Reclaimable - Single department", func() {
	var (
		reclaimerInfo *ReclaimerInfo
		reclaimees    []*podgroup_info.PodGroupInfo
		queues        map[common_info.QueueID]*rs.QueueAttributes
		reclaimable   *Reclaimable
	)
	BeforeEach(func() {
		reclaimerInfo = &ReclaimerInfo{
			Name:              "reclaimer",
			Namespace:         "n1",
			Queue:             "p1",
			IsPreemptable:     true,
			RequiredResources: resource_info.NewResource(0, 0, 1).ToVector(testVectorMap),
			VectorMap:         testVectorMap,
		}

		reclaimee := testReclaimee("reclaimee", "p2", pod_info.PodsMap{
			"1": testPodInfo("1", 1, pod_status.Running),
		})
		reclaimees = []*podgroup_info.PodGroupInfo{reclaimee}

		queues = map[common_info.QueueID]*rs.QueueAttributes{
			"p1": {
				UID:         "p1",
				Name:        "p1",
				ParentQueue: "default",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:                3,
						FairShare:               3,
						Allocated:               2,
						AllocatedNotPreemptible: 0,
						MaxAllowed:              commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
			"p2": {
				UID:         "p2",
				Name:        "p2",
				ParentQueue: "default",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:                2,
						FairShare:               2,
						Allocated:               3,
						AllocatedNotPreemptible: 0,
						MaxAllowed:              commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
			"default": {
				UID:  "default",
				Name: "default",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:                5,
						FairShare:               5,
						Allocated:               5,
						AllocatedNotPreemptible: 0,
						MaxAllowed:              commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
		}
		reclaimable = New(1.0)
	})
	It("Reclaimer is below fair share, reclaimee above fair share", func() {
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimer is below fair share, reclaimer exactly at fair share", func() {
		queues["p2"].GPU.Allocated = 2
		queues["default"].GPU.Allocated = 4
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer and reclaimee are below fair share", func() {
		queues["p2"].GPU.Allocated = 1
		queues["default"].GPU.Allocated = 3
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer below deserved and reclaimee above deserved (within fair share)", func() {
		//Set fair share to 3
		queues["p2"].QueueResourceShare.AddResourceShare(rs.GpuResource, 3-queues["p2"].GPU.FairShare)
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimer at fair share, reclaimee above fair share, department below fair share", func() {
		queues["p1"].GPU.Allocated = 3
		queues["default"].GPU.Allocated = 6
		queues["default"].GPU.Deserved = 7
		queues["default"].QueueResourceShare.AddResourceShare(rs.GpuResource, 7-queues["p2"].GPU.FairShare)
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer at fair share, reclaimee above fair share, department above fair share", func() {
		queues["p1"].GPU.Allocated = 3
		queues["default"].GPU.Allocated = 6
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer above deserved, attempting to reclaim for non preemptible job", func() {
		queues["p1"].GPU.Deserved = 2
		reclaimerInfo.IsPreemptable = false
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimer department only preemptible above deserved, attempting to reclaim for non preemptible job", func() {
		queues["p1"].GPU.Deserved = 2
		reclaimerInfo.IsPreemptable = false
		queues["default"].GPU.Deserved = 3
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimer department nonpreemtible equal to deserved, attempting to reclaim for non preemptible job", func() {
		queues["p1"].GPU.Deserved = 2
		reclaimerInfo.IsPreemptable = false
		queues["default"].GPU.Deserved = 3
		queues["default"].GPU.AllocatedNotPreemptible = 3
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer department allocated above fair share, attempting to reclaim for job", func() {
		queues["p1"].GPU.Deserved = 2
		queues["p1"].GPU.Deserved = 2
		reclaimerInfo.IsPreemptable = true
		queues["default"].GPU.Deserved = 1
		queues["default"].GPU.FairShare = 1
		queues["default"].GPU.Allocated = 3
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
})

var _ = Describe("Reclaimable - Multiple departments", func() {
	var (
		reclaimerInfo *ReclaimerInfo
		reclaimees    []*podgroup_info.PodGroupInfo
		queues        map[common_info.QueueID]*rs.QueueAttributes
		reclaimable   *Reclaimable
	)
	BeforeEach(func() {
		reclaimerInfo = &ReclaimerInfo{
			Name:              "reclaimer",
			Namespace:         "n1",
			Queue:             "p1",
			IsPreemptable:     true,
			RequiredResources: resource_info.NewResource(0, 0, 1).ToVector(testVectorMap),
			VectorMap:         testVectorMap,
		}

		reclaimee := testReclaimee("reclaimee", "p2", pod_info.PodsMap{
			"1": testPodInfo("1", 1, pod_status.Running),
		})
		reclaimees = []*podgroup_info.PodGroupInfo{reclaimee}

		queues = map[common_info.QueueID]*rs.QueueAttributes{
			"p1": {
				UID:         "p1",
				Name:        "p1",
				ParentQueue: "d1",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:   3,
						FairShare:  3,
						Allocated:  2,
						MaxAllowed: commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
			"p2": {
				UID:         "p2",
				Name:        "p2",
				ParentQueue: "d2",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:   2,
						FairShare:  2,
						Allocated:  3,
						MaxAllowed: commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
			"d1": {
				UID:  "d1",
				Name: "d1",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:   3,
						FairShare:  3,
						Allocated:  2,
						MaxAllowed: commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
			"d2": {
				UID:  "d2",
				Name: "d2",
				QueueResourceShare: rs.QueueResourceShare{
					GPU: rs.ResourceShare{
						Deserved:   2,
						FairShare:  2,
						Allocated:  3,
						MaxAllowed: commonconstants.UnlimitedResourceQuantity,
					},
					CPU:    rs.ResourceShare{},
					Memory: rs.ResourceShare{},
				},
			},
		}
		reclaimable = New(1.0)
	})
	It("Reclaimer is below fair share, reclaimee above fair share - sanity", func() {
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimee department goes below fair share", func() {
		queues["p2"].GPU.Allocated = 2
		queues["d2"].GPU.Allocated = 2
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer department is below deserved and reclaimee department is above deserved but within fair share", func() {
		queues["p1"].GPU.Allocated = 1
		queues["d1"].GPU.Allocated = 1
		queues["p2"].GPU.FairShare = 4
		queues["d2"].GPU.FairShare = 4
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
})

var _ = Describe("Reclaimable - Multiple hierarchy levels", func() {
	var (
		reclaimerInfo *ReclaimerInfo
		reclaimee     *podgroup_info.PodGroupInfo
		queuesData    map[common_info.QueueID]queuesTestData
		reclaimable   *Reclaimable
	)
	BeforeEach(func() {
		reclaimerInfo = &ReclaimerInfo{
			Name:              "reclaimer",
			Namespace:         "n1",
			Queue:             "left-leaf",
			IsPreemptable:     true,
			RequiredResources: resource_info.NewResource(0, 0, 1).ToVector(testVectorMap),
			VectorMap:         testVectorMap,
		}

		reclaimee = testReclaimee("reclaimee", "right-leaf", pod_info.PodsMap{
			"1": testPodInfo("1", 2, pod_status.Running),
		})
	})
	It("Reclaimer is below fair share, reclaimee above fair share - sanity", func() {
		queuesData = map[common_info.QueueID]queuesTestData{
			"left-top": {
				"",
				1,
				1,
				0,
			},
			"left-mid": {
				"left-top",
				1,
				1,
				0,
			},
			"left-leaf": {
				"left-mid",
				1,
				1,
				0,
			},

			"right-top": {
				"",
				1,
				1,
				2,
			},
			"right-mid": {
				"right-top",
				1,
				1,
				2,
			},
			"right-leaf": {
				"right-mid",
				1,
				1,
				2,
			},
		}
		queues := buildQueues(queuesData)
		reclaimable = New(1.0)
		reclaimees := []*podgroup_info.PodGroupInfo{reclaimee}
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimer top queue will go over quota - don't reclaim", func() {
		queuesData = map[common_info.QueueID]queuesTestData{
			"left-top": {
				"",
				1,
				1,
				1,
			},
			"left-top-oq-leaf": {
				"left-top",
				0,
				0,
				1,
			},
			"left-mid": {
				"left-top",
				1,
				1,
				0,
			},
			"left-leaf": {
				"left-mid",
				1,
				1,
				0,
			},

			"right-top": {
				"",
				1,
				1,
				2,
			},
			"right-mid": {
				"right-top",
				1,
				1,
				2,
			},
			"right-leaf": {
				"right-mid",
				1,
				1,
				2,
			},
		}
		queues := buildQueues(queuesData)
		reclaimable = New(1.0)
		reclaimees := []*podgroup_info.PodGroupInfo{reclaimee}
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer in the same tree branch and will go over fair share - don't reclaim", func() {
		queuesData = map[common_info.QueueID]queuesTestData{
			"top": {
				"",
				2,
				2,
				2,
			},
			"mid1": {
				"top",
				1,
				1,
				0.5,
			},
			"mid2": {
				"top",
				1,
				1,
				1.5,
			},
			"left-leaf1": {
				"mid1",
				1,
				1,
				0,
			},
			"left-leaf2": {
				"mid1",
				0,
				0,
				0.5,
			},
			"right-leaf": {
				"mid2",
				1,
				1,
				1.5,
			},
		}
		queues := buildQueues(queuesData)
		reclaimable = New(1.0)

		pod := reclaimee.GetAllPodsMap()["1"]
		pod.GpuRequirement = *resource_info.NewGpuResourceRequirementWithGpus(1.5, 0)
		pod.ResReqVector = resource_info.NewResourceVectorWithValues(0, 0, 1.5, testVectorMap)
		reclaimerInfo.RequiredResources = resource_info.NewResource(0, 0, 1).ToVector(testVectorMap)
		reclaimerInfo.Queue = "left-leaf1"

		reclaimees := []*podgroup_info.PodGroupInfo{reclaimee}
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(false))
	})
	It("Reclaimer in the same tree branch - reclaim", func() {
		queuesData = map[common_info.QueueID]queuesTestData{
			"top": {
				"",
				2,
				2,
				2,
			},
			"mid1": {
				"top",
				1,
				1,
				0.5,
			},
			"mid2": {
				"top",
				1,
				1,
				1.5,
			},
			"left-leaf1": {
				"mid1",
				1,
				1,
				0,
			},
			"left-leaf2": {
				"mid1",
				0,
				0,
				0.5,
			},
			"right-leaf": {
				"mid2",
				1,
				1,
				1.5,
			},
		}
		queues := buildQueues(queuesData)
		reclaimable = New(1.0)

		reclaimerInfo.RequiredResources = resource_info.NewResource(0, 0, 1).ToVector(testVectorMap)
		reclaimerInfo.Queue = "left-leaf1"
		pod := reclaimee.GetAllPodsMap()["1"]
		pod.GpuRequirement = *resource_info.NewGpuResourceRequirementWithGpus(1.5, 0)
		pod.ResReqVector = resource_info.NewResourceVectorWithValues(0, 0, 1.5, testVectorMap)
		reclaimee2 := testReclaimee("reclaimee", "left-leaf2", pod_info.PodsMap{
			"1": testPodInfo("1", 0.5, pod_status.Running),
		})

		reclaimees := []*podgroup_info.PodGroupInfo{reclaimee, reclaimee2}
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclaimer has lower utilization ratio than reclaimee but over 1", func() {
		queuesData = map[common_info.QueueID]queuesTestData{
			"d1": {
				"",
				4,
				4,
				4,
			},
			"d1-project-1": {
				"d1",
				3,
				1,
				0,
			},
			"d1-project-2": {
				"d1",
				1,
				3,
				4,
			},
			"d2": {
				"",
				3,
				3,
				7,
			},
			"d2-project-1": {
				"d2",
				3,
				3,
				7,
			},
		}
		queues := buildQueues(queuesData)
		reclaimable = New(1.0)

		reclaimerInfo.RequiredResources = resource_info.NewResource(0, 0, 1).ToVector(testVectorMap)
		reclaimerInfo.Queue = "d1-project-1"
		reclaimee2 := testReclaimee("reclaimee2", "d2-project-1", pod_info.PodsMap{
			"1": testPodInfo("1", 1, pod_status.Running),
		})

		reclaimees := []*podgroup_info.PodGroupInfo{reclaimee2}
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
	It("Reclamation with uninvolved resources", func() {
		queuesData = map[common_info.QueueID]queuesTestData{
			"d1": {
				"",
				4,
				4,
				4,
			},
			"d1-project-1": {
				"d1",
				1,
				1,
				1,
			},
			"d2": {
				"",
				3,
				3,
				7,
			},
			"d2-project-1": {
				"d2",
				3,
				3,
				7,
			},
		}
		queues := buildQueues(queuesData)
		// Set high CPU allocation for d1 to create a high utilization ratio
		queues["d1"].CPU.Allocated = 3000
		queues["d1"].CPU.FairShare = 1000 // This creates a 3.0 utilization ratio
		queues["d2"].CPU.Allocated = 1000
		queues["d2"].CPU.FairShare = 1000 // This creates a 1.0 utilization ratio

		reclaimable = New(1.0)

		reclaimerInfo.RequiredResources = resource_info.NewResource(0, 0, 1).ToVector(testVectorMap) // Only requests GPU
		reclaimerInfo.Queue = "d1-project-1"
		reclaimee2 := testReclaimee("reclaimee2", "d2-project-1", pod_info.PodsMap{
			"1": testPodInfo("1", 1, pod_status.Running),
		})

		reclaimees := []*podgroup_info.PodGroupInfo{reclaimee2}
		result := reclaimable.Reclaimable(queues, reclaimerInfo, reclaimeeResourcesByQueue(reclaimees))
		Expect(result).To(Equal(true))
	})
})

var _ = Describe("FilterVictim", func() {
	reclaimerInfo := &ReclaimerInfo{
		Queue:             "reclaimer",
		RequiredResources: resource_info.NewResource(0, 0, 2).ToVector(testVectorMap),
		VectorMap:         testVectorMap,
		IsPreemptable:     true,
	}

	It("filters victims whose leveled queue is strictly under deserved quota", func() {
		reclaimable := New(1)
		queues := buildQueues(map[common_info.QueueID]queuesTestData{
			"reclaimer": {deserved: 4, fairShare: 4, allocated: 0},
			"victim":    {deserved: 4, fairShare: 4, allocated: 2},
		})

		Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "victim")).To(BeFalse())
	})

	It("keeps victims whose leveled queue is over deserved quota", func() {
		reclaimable := New(1)
		queues := buildQueues(map[common_info.QueueID]queuesTestData{
			"reclaimer": {deserved: 4, fairShare: 4, allocated: 0},
			"victim":    {deserved: 0, fairShare: 4, allocated: 4},
		})

		Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "victim")).To(BeTrue())
	})

	It("keeps victims exactly at deserved quota for consolidation", func() {
		reclaimable := New(1)
		queues := buildQueues(map[common_info.QueueID]queuesTestData{
			"reclaimer": {deserved: 4, fairShare: 4, allocated: 0},
			"victim":    {deserved: 2, fairShare: 4, allocated: 2},
		})

		Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "victim")).To(BeTrue())
	})

	It("filters victims under allocatable share when the reclaimer cannot use deserved quota", func() {
		reclaimable := New(1)
		queues := buildQueues(map[common_info.QueueID]queuesTestData{
			"reclaimer": {deserved: 1, fairShare: 4, allocated: 0},
			"victim":    {deserved: 2, fairShare: 4, allocated: 3},
		})

		Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "victim")).To(BeFalse())
	})

	It("keeps victims over allocatable share when the reclaimer cannot use deserved quota", func() {
		reclaimable := New(1)
		queues := buildQueues(map[common_info.QueueID]queuesTestData{
			"reclaimer": {deserved: 1, fairShare: 4, allocated: 0},
			"victim":    {deserved: 2, fairShare: 4, allocated: 5},
		})

		Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "victim")).To(BeTrue())
	})
})

var _ = Describe("Reclaimable - displacing lower priority jobs of the reclaimer queue", func() {
	var (
		reclaimerInfo *ReclaimerInfo
		queues        map[common_info.QueueID]*rs.QueueAttributes
		reclaimable   *Reclaimable
	)
	otherQueueVictim := resource_info.NewResource(16, 0, 2).ToVector(testVectorMap)

	BeforeEach(func() {
		// research is under its GPU fair share and over its CPU fair share only because of its own lower-priority CPU
		// jobs, while posttrain is over its GPU fair share. posttrain is also more CPU-saturated than research, so the
		// sibling saturation check alone does not reject scenarios that leave research over its fair share.
		reclaimerInfo = &ReclaimerInfo{
			Name:              "reclaimer",
			Namespace:         "n1",
			Queue:             "research",
			IsPreemptable:     true,
			RequiredResources: resource_info.NewResource(16, 0, 2).ToVector(testVectorMap),
			VectorMap:         testVectorMap,
		}
		queues = map[common_info.QueueID]*rs.QueueAttributes{
			"research":  buildCPUAndGPUQueue("research", "rhoda", resourceTestData{160, 160, 400}, resourceTestData{8, 8, 0}),
			"posttrain": buildCPUAndGPUQueue("posttrain", "rhoda", resourceTestData{160, 160, 1000}, resourceTestData{160, 160, 260}),
			"rhoda":     buildCPUAndGPUQueue("rhoda", "", resourceTestData{320, 320, 1400}, resourceTestData{168, 168, 260}),
		}
		reclaimable = New(1.0)
	})

	Context("CanReclaimResources", func() {
		It("refuses when the queue is over its fair share and nothing is displaceable", func() {
			Expect(reclaimable.CanReclaimResources(queues, reclaimerInfo)).To(BeFalse())
		})
		It("allows when displacing the queue's lower priority jobs brings it within its fair share", func() {
			reclaimerInfo.Displaceable = rs.NewResourceQuantities(384, 0, 0)
			Expect(reclaimable.CanReclaimResources(queues, reclaimerInfo)).To(BeTrue())
		})
		It("refuses when displacing the queue's lower priority jobs is not enough to fit its fair share", func() {
			reclaimerInfo.Displaceable = rs.NewResourceQuantities(200, 0, 0)
			Expect(reclaimable.CanReclaimResources(queues, reclaimerInfo)).To(BeFalse())
		})
	})

	Context("Reclaimable", func() {
		BeforeEach(func() {
			reclaimerInfo.Displaceable = rs.NewResourceQuantities(384, 0, 0)
		})
		It("accepts a scenario evicting own-queue jobs and another queue's over fair share job", func() {
			victims := map[common_info.QueueID][]resource_info.ResourceVector{
				"research":  {resource_info.NewResource(384, 0, 0).ToVector(testVectorMap)},
				"posttrain": {otherQueueVictim},
			}
			Expect(reclaimable.Reclaimable(queues, reclaimerInfo, victims)).To(BeTrue())
		})
		It("rejects a scenario that keeps the own-queue jobs running", func() {
			victims := map[common_info.QueueID][]resource_info.ResourceVector{
				"posttrain": {otherQueueVictim},
			}
			Expect(reclaimable.Reclaimable(queues, reclaimerInfo, victims)).To(BeFalse())
		})
		It("rejects a scenario whose own-queue evictions leave the queue over its fair share", func() {
			victims := map[common_info.QueueID][]resource_info.ResourceVector{
				"research":  {resource_info.NewResource(200, 0, 0).ToVector(testVectorMap)},
				"posttrain": {otherQueueVictim},
			}
			Expect(reclaimable.Reclaimable(queues, reclaimerInfo, victims)).To(BeFalse())
		})
	})

	Context("FilterVictim", func() {
		It("keeps own-queue victims when the queue must displace jobs to fit its fair share", func() {
			Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "research")).To(BeTrue())
		})
		It("filters own-queue victims when the queue fits its fair share without them", func() {
			queues["research"].CPU.Allocated = 0
			Expect(reclaimable.FilterVictim(queues, reclaimerInfo, "research")).To(BeFalse())
		})
	})
})

type resourceTestData struct {
	deserved  float64
	fairShare float64
	allocated float64
}

func buildCPUAndGPUQueue(uid, parent common_info.QueueID, cpu, gpu resourceTestData) *rs.QueueAttributes {
	return &rs.QueueAttributes{
		UID:         uid,
		Name:        string(uid),
		ParentQueue: parent,
		QueueResourceShare: rs.QueueResourceShare{
			GPU: rs.ResourceShare{
				Deserved:   gpu.deserved,
				FairShare:  gpu.fairShare,
				Allocated:  gpu.allocated,
				MaxAllowed: commonconstants.UnlimitedResourceQuantity,
			},
			CPU: rs.ResourceShare{
				Deserved:   cpu.deserved,
				FairShare:  cpu.fairShare,
				Allocated:  cpu.allocated,
				MaxAllowed: commonconstants.UnlimitedResourceQuantity,
			},
			Memory: rs.ResourceShare{},
		},
	}
}

func buildQueues(queuesData map[common_info.QueueID]queuesTestData) map[common_info.QueueID]*rs.QueueAttributes {
	queues := map[common_info.QueueID]*rs.QueueAttributes{}
	for name, queueData := range queuesData {
		queues[name] = &rs.QueueAttributes{
			UID:         name,
			Name:        string(name),
			ParentQueue: queueData.parentQueue,
			QueueResourceShare: rs.QueueResourceShare{
				GPU: rs.ResourceShare{
					Deserved:   queueData.deserved,
					FairShare:  queueData.fairShare,
					Allocated:  queueData.allocated,
					MaxAllowed: commonconstants.UnlimitedResourceQuantity,
				},
				CPU:    rs.ResourceShare{},
				Memory: rs.ResourceShare{},
			},
		}
	}
	return queues
}

func reclaimeeResourcesByQueue(reclaimees []*podgroup_info.PodGroupInfo) map[common_info.QueueID][]resource_info.ResourceVector {
	resources := make(map[common_info.QueueID][]resource_info.ResourceVector)
	for _, reclaimee := range reclaimees {

		if _, found := resources[reclaimee.Queue]; !found {
			resources[reclaimee.Queue] = make([]resource_info.ResourceVector, 0)
		}
		resources[reclaimee.Queue] = append(resources[reclaimee.Queue], reclaimee.GetTasksActiveAllocatedReqResourceVector())
	}

	return resources
}
