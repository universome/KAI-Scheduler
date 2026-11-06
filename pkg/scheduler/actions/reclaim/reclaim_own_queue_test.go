// Copyright 2025 NVIDIA CORPORATION
// SPDX-License-Identifier: Apache-2.0

package reclaim_test

import (
	"testing"

	. "go.uber.org/mock/gomock"
	"gopkg.in/h2non/gock.v1"

	enginev2alpha2 "github.com/kai-scheduler/KAI-scheduler/pkg/apis/scheduling/v2alpha2"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/actions/integration_tests/integration_tests_utils"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/actions/reclaim"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/api/pod_status"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/constants"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/test_utils"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/test_utils/jobs_fake"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/test_utils/nodes_fake"
	"github.com/kai-scheduler/KAI-scheduler/pkg/scheduler/test_utils/tasks_fake"
)

func TestHandleReclaimWithOwnQueueDisplacement(t *testing.T) {
	test_utils.InitTestingInfrastructure()
	controller := NewController(t)
	defer controller.Finish()
	defer gock.Off()

	for testNumber, testMetadata := range getOwnQueueDisplacementTestsMetadata() {
		t.Logf("Running test number: %v, test name: %v,", testNumber, testMetadata.TestTopologyBasic.Name)
		ssn := test_utils.BuildSession(testMetadata.TestTopologyBasic, controller)
		reclaimAction := reclaim.New()
		reclaimAction.Execute(ssn)

		test_utils.MatchExpectedAndRealTasks(t, testNumber, testMetadata.TestTopologyBasic, ssn)
	}
}

// Queue CPU quotas are in millicores, while job and node CPUs in these fixtures are in cores.
const milliCPUsPerCPU = 1000

type ownQueueDisplacementCase struct {
	name                  string
	cpuJob                *jobs_fake.TestJobBasic
	nodes                 map[string]nodes_fake.TestNodeBasic
	posttrainDeservedCPUs float64
	// cpuSinkDeservedCPUs, when set, adds a queue with an unschedulable pending CPU job that absorbs the CPU left
	// over by research and posttrain quotas, so that both fair shares equal their quotas.
	cpuSinkDeservedCPUs float64
	expectedResults     map[string]test_utils.TestExpectedResultBasic
	cacheRequirements   *test_utils.CacheMocking
}

// ownQueueDisplacementTopology has a research queue that is under its GPU fair share but over its CPU fair share
// because of its own running CPU-only job, and a posttrain queue that is over its GPU fair share. The pending posttrain
// CPU job raises posttrain's CPU fair share so that research's CPU fair share equals its quota.
func ownQueueDisplacementTopology(testCase ownQueueDisplacementCase) integration_tests_utils.TestTopologyMetadata {
	jobs := []*jobs_fake.TestJobBasic{
		{
			Name:                "pt_job0",
			RequiredGPUsPerTask: 2,
			RequiredCPUsPerTask: 4000,
			Priority:            constants.PriorityTrainNumber,
			QueueName:           "posttrain",
			Tasks:               []*tasks_fake.TestTaskBasic{{NodeName: "node0", State: pod_status.Running}},
		},
		{
			Name:                "pt_job1",
			RequiredGPUsPerTask: 2,
			RequiredCPUsPerTask: 4000,
			Priority:            constants.PriorityTrainNumber - 1,
			QueueName:           "posttrain",
			Tasks:               []*tasks_fake.TestTaskBasic{{NodeName: "node0", State: pod_status.Running}},
		},
		{
			Name:                "pt_pending_cpu",
			RequiredCPUsPerTask: 16000,
			Priority:            constants.PriorityTrainNumber,
			QueueName:           "posttrain",
			Tasks:               []*tasks_fake.TestTaskBasic{{State: pod_status.Pending}},
		},
		testCase.cpuJob,
		{
			Name:                "reclaimer",
			RequiredGPUsPerTask: 2,
			RequiredCPUsPerTask: 8000,
			Priority:            constants.PriorityTrainNumber,
			QueueName:           "research",
			Tasks:               []*tasks_fake.TestTaskBasic{{State: pod_status.Pending}},
		},
	}
	queues := []test_utils.TestQueueBasic{
		{
			Name:         "research",
			DeservedGPUs: 2,
			DeservedCPUs: test_utils.CreateFloat64Pointer(8000 * milliCPUsPerCPU),
		},
		{
			Name:         "posttrain",
			DeservedGPUs: 2,
			DeservedCPUs: test_utils.CreateFloat64Pointer(testCase.posttrainDeservedCPUs * milliCPUsPerCPU),
		},
	}
	if testCase.cpuSinkDeservedCPUs > 0 {
		jobs = append(jobs, &jobs_fake.TestJobBasic{
			Name:                "cpu_sink_pending",
			RequiredCPUsPerTask: testCase.cpuSinkDeservedCPUs,
			Priority:            constants.PriorityTrainNumber,
			QueueName:           "cpu_sink",
			Tasks:               []*tasks_fake.TestTaskBasic{{State: pod_status.Pending}},
		})
		queues = append(queues, test_utils.TestQueueBasic{
			Name:         "cpu_sink",
			DeservedCPUs: test_utils.CreateFloat64Pointer(testCase.cpuSinkDeservedCPUs * milliCPUsPerCPU),
		})
	}

	return integration_tests_utils.TestTopologyMetadata{
		TestTopologyBasic: test_utils.TestTopologyBasic{
			Name:               testCase.name,
			Jobs:               jobs,
			Nodes:              testCase.nodes,
			Queues:             queues,
			JobExpectedResults: testCase.expectedResults,
			Mocks:              &test_utils.TestMock{CacheRequirements: testCase.cacheRequirements},
		},
	}
}

func researchCPUJob(priority int, preemptibility enginev2alpha2.Preemptibility, nodeName string) *jobs_fake.TestJobBasic {
	return &jobs_fake.TestJobBasic{
		Name:                "research_cpu_job",
		RequiredCPUsPerTask: 24000,
		Priority:            int32(priority),
		Preemptibility:      preemptibility,
		QueueName:           "research",
		Tasks:               []*tasks_fake.TestTaskBasic{{NodeName: nodeName, State: pod_status.Running}},
	}
}

func unchangedOwnQueueDisplacementResults(cpuJobNodeName string) map[string]test_utils.TestExpectedResultBasic {
	return map[string]test_utils.TestExpectedResultBasic{
		"pt_job0":          {NodeName: "node0", GPUsRequired: 2, Status: pod_status.Running},
		"pt_job1":          {NodeName: "node0", GPUsRequired: 2, Status: pod_status.Running},
		"pt_pending_cpu":   {Status: pod_status.Pending},
		"research_cpu_job": {NodeName: cpuJobNodeName, Status: pod_status.Running},
		"reclaimer":        {GPUsRequired: 2, Status: pod_status.Pending},
	}
}

func getOwnQueueDisplacementTestsMetadata() []integration_tests_utils.TestTopologyMetadata {
	singleNode := map[string]nodes_fake.TestNodeBasic{
		"node0": {GPUs: 4, CPUMillis: 32000},
	}
	gpuAndCPUNodes := map[string]nodes_fake.TestNodeBasic{
		"node0": {GPUs: 4, CPUMillis: 16000},
		"node1": {CPUMillis: 24000},
	}
	noCacheOperations := &test_utils.CacheMocking{}

	return []integration_tests_utils.TestTopologyMetadata{
		ownQueueDisplacementTopology(ownQueueDisplacementCase{
			name:                  "Reclaim by displacing own lower priority jobs and another queue's over fair share job",
			cpuJob:                researchCPUJob(constants.PriorityTrainNumber-10, "", "node0"),
			nodes:                 singleNode,
			posttrainDeservedCPUs: 24000,
			expectedResults: map[string]test_utils.TestExpectedResultBasic{
				"pt_job0":          {NodeName: "node0", GPUsRequired: 2, Status: pod_status.Running},
				"pt_job1":          {NodeName: "node0", GPUsRequired: 2, Status: pod_status.Releasing},
				"pt_pending_cpu":   {Status: pod_status.Pending},
				"research_cpu_job": {NodeName: "node0", Status: pod_status.Releasing},
				"reclaimer":        {NodeName: "node0", GPUsRequired: 2, Status: pod_status.Pipelined},
			},
			cacheRequirements: &test_utils.CacheMocking{NumberOfCacheEvictions: 2, NumberOfPipelineActions: 1},
		}),
		ownQueueDisplacementTopology(ownQueueDisplacementCase{
			// posttrain is more CPU-saturated than research would be, so only the reclaimer queue's own fair share
			// check rejects scenarios in which the re-placed own job keeps research over its fair share.
			name:                  "Don't reclaim when the own lower priority jobs would be re-placed and the queue would stay over fair share",
			cpuJob:                researchCPUJob(constants.PriorityTrainNumber-10, "", "node1"),
			nodes:                 gpuAndCPUNodes,
			posttrainDeservedCPUs: 500,
			cpuSinkDeservedCPUs:   31500,
			expectedResults:       unchangedOwnQueueDisplacementResults("node1"),
			cacheRequirements:     noCacheOperations,
		}),
		ownQueueDisplacementTopology(ownQueueDisplacementCase{
			name:                  "Don't displace own jobs with a higher priority than the reclaimer",
			cpuJob:                researchCPUJob(constants.PriorityTrainNumber+10, "", "node0"),
			nodes:                 singleNode,
			posttrainDeservedCPUs: 24000,
			expectedResults:       unchangedOwnQueueDisplacementResults("node0"),
			cacheRequirements:     noCacheOperations,
		}),
		ownQueueDisplacementTopology(ownQueueDisplacementCase{
			name:                  "Don't displace own non-preemptible jobs",
			cpuJob:                researchCPUJob(constants.PriorityTrainNumber-10, enginev2alpha2.NonPreemptible, "node0"),
			nodes:                 singleNode,
			posttrainDeservedCPUs: 24000,
			expectedResults:       unchangedOwnQueueDisplacementResults("node0"),
			cacheRequirements:     noCacheOperations,
		}),
	}
}
