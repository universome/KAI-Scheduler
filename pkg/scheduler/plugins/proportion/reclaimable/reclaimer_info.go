// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package reclaimable

import (
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/common_info"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/resource_info"
	rs "github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/plugins/proportion/resource_share"
)

type ReclaimerInfo struct {
	Name              string
	Namespace         string
	Queue             common_info.QueueID
	RequiredResources resource_info.ResourceVector
	VectorMap         *resource_info.ResourceVectorMap
	IsPreemptable     bool
	// Displaceable holds the resources of running, preemptible, lower-priority jobs of the reclaimer's own queue,
	// which a reclaim scenario may evict together with victims from other queues.
	Displaceable rs.ResourceQuantities
}
