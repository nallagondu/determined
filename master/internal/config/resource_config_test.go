//nolint:exhaustruct
package config

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestResourceManagers(t *testing.T) {
	r := ResourceConfig{
		RootManagerInternal: &ResourceManagerConfig{
			KubernetesRM: &KubernetesResourceManagerConfig{},
		},
		RootPoolsInternal: []ResourcePoolConfig{
			{
				PoolName: "root",
			},
		},
		AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
			{
				ResourceManager: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{},
				},
				ResourcePools: []ResourcePoolConfig{
					{
						PoolName: "test",
					},
				},
			},
		},
	}

	expected := []*ResourceManagerWithPoolsConfig{
		{
			ResourceManager: &ResourceManagerConfig{
				KubernetesRM: &KubernetesResourceManagerConfig{},
			},
			ResourcePools: []ResourcePoolConfig{
				{
					PoolName: "root",
				},
			},
		},
		{
			ResourceManager: &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{},
			},
			ResourcePools: []ResourcePoolConfig{
				{
					PoolName: "test",
				},
			},
		},
	}
	require.Equal(t, expected, r.ResourceManagers())
}

func TestGetKubernetesClusterNames(t *testing.T) {
	cases := []struct {
		name          string
		config        ResourceConfig
		expectedNames []string
	}{
		{
			"agent rm",
			ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{},
				},
			},
			[]string{},
		},
		{
			"dispatcher rm",
			ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					DispatcherRM: &DispatcherResourceManagerConfig{},
				},
			},
			[]string{},
		},
		{
			"kubernetes rm",
			ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					KubernetesRM: &KubernetesResourceManagerConfig{
						ClusterName: "test",
					},
				},
			},
			[]string{"test"},
		},
		{
			"multi rm",
			ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					KubernetesRM: &KubernetesResourceManagerConfig{
						ClusterName: "test1",
					},
				},
				AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
					{
						ResourceManager: &ResourceManagerConfig{
							KubernetesRM: &KubernetesResourceManagerConfig{
								ClusterName: "test2",
							},
						},
					},
				},
			},
			[]string{"test1", "test2"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			names := c.config.GetKubernetesClusterNames()
			require.Equal(t, c.expectedNames, names)
		})
	}
}

func TestGetAgentRMConfig(t *testing.T) {
	t.Run("no agent rm", func(t *testing.T) {
		noAgentRM := ResourceConfig{
			RootManagerInternal: &ResourceManagerConfig{
				KubernetesRM: &KubernetesResourceManagerConfig{},
			},
			AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
				{
					ResourceManager: &ResourceManagerConfig{
						KubernetesRM: &KubernetesResourceManagerConfig{},
					},
				},
			},
		}

		conf, ok := noAgentRM.GetAgentRMConfig()
		require.False(t, ok)
		require.Nil(t, conf)
	})

	t.Run("has agent rm", func(t *testing.T) {
		hasAgentRM := ResourceConfig{
			RootManagerInternal: &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{},
			},
			AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
				{
					ResourceManager: &ResourceManagerConfig{
						KubernetesRM: &KubernetesResourceManagerConfig{},
					},
				},
			},
		}

		conf, ok := hasAgentRM.GetAgentRMConfig()
		require.True(t, ok)
		require.Equal(t, hasAgentRM.ResourceManagers()[0], conf)
	})
}

func TestResolveConfigErrors(t *testing.T) {
	cases := []struct {
		name                  string
		yaml                  string
		expectedError         error
		validationErrorString string
	}{
		// TODO(RM-XXX) why is "Check Failed 2" errors.
		// I think it is because of check.Validate calling it twice somehow.
		{
			"dupe pools", `
resource_manager:
  type: agent
  name: a
  cluster_name: a
resource_pools:
  - pool_name: a
  - pool_name: a`, nil, "Check Failed! 2 errors found:\n\terror found at root.ResourceConfig: " +
				"resource pool has a duplicate name: a\n\terror found at root: resource pool has " +
				"a duplicate name: a",
		},

		{
			"dupe pools in additional rm", `
resource_manager:
  type: agent
  name: a
  cluster_name: a
resource_pools:
  - pool_name: a
additional_resource_managers:
  - resource_manager:
      type: kubernetes
      max_slots_per_pod: 2
      name: b
      cluster_name: b
    resource_pools:
      - pool_name: a`, nil, "Check Failed! 2 errors found:\n\terror found at root.ResourceConfig: " +
				"resource pool has a duplicate name: a They must be unique across even different " +
				"resource managers\n\terror found at root: resource pool has a duplicate name: a " +
				"They must be unique across even different resource managers",
		},

		{"dupe rm names", `
resource_manager:
  type: agent
  name: a
  cluster_name: a
additional_resource_managers:
  - resource_manager:
      type: kubernetes
      name: a
      cluster_name: a
      max_slots_per_pod: 2
    resource_pools:
    - pool_name: a`, nil, "Check Failed! 2 errors found:\n\terror found at " +
			"root.ResourceConfig: resource manager has a duplicate name: a\n\terror " +
			"found at root: resource manager has a duplicate name: a"},

		{"more than one agent", `
resource_manager:
  type: agent
  name: a
  cluster_name: a
additional_resource_managers:
  - resource_manager:
      type: agent
      name: b
      cluster_name: b
    resource_pools:
    - pool_name: a`, nil, "Check Failed! 2 errors found:\n\terror found at root.ResourceConfig: " +
			"additional_resource_managers only supports resource managers of type: " +
			"kubernetes\n\terror found at root: additional_resource_managers only supports " +
			"resource managers of type: kubernetes"},

		{"k8s name not specified", `
resource_manager:
  type: agent
  name: a
  cluster_name: a
additional_resource_managers:
  - resource_manager:
      type: kubernetes
      max_slots_per_pod: 12
    resource_pools:
    - pool_name: a`, nil, "Check Failed! 1 errors found:\n\terror found at " +
			"root.ResourceConfig.AdditionalResourceManagersInternal[0]." +
			"ResourceManager.KubernetesRM: cluster_name is required:  must be non-empty"},

		{"additional rm not giving pools", `
resource_manager:
  type: agent
  name: a
  cluster_name: a
additional_resource_managers:
  - resource_manager:
      type: kubernetes
      name: test
      cluster_name: test
      max_slots_per_pod: 12`, nil, "Check Failed! 2 errors found:\n\terror found at " +
			"root.ResourceConfig: for additional_resource_managers, you must specify at " +
			"least one resource pool\n\terror found at root: for additional_resource_managers, " +
			"you must specify at least one resource pool"},

		{"k8s rocm config", `
resource_manager:
  type: kubernetes
  max_slots_per_pod: 1
  name: a
  cluster_name: a
  slot_type: rocm`, nil, "Check Failed! 1 errors found:\n\terror found at root.ResourceConfig." +
			"RootManagerInternal.KubernetesRM: rocm slot_type is not supported yet on k8s"},

		{"k8s negative cpu", `
resource_manager:
  type: kubernetes
  max_slots_per_pod: 1
  name: a
  cluster_name: a
  slot_type: cpu
  slot_resource_requests:
    cpu: -10`, nil, "Check Failed! 1 errors found:\n\terror found at root.ResourceConfig." +
			"RootManagerInternal.KubernetesRM: slot_resource_requests.cpu " +
			"must be > 0: -10 is not greater than 0"},

		{"k8s missing gateway_name", `
resource_manager:
  type: kubernetes
  max_slots_per_pod: 1
  name: a
  internal_task_gateway:
    gateway_namespace: test
    gateway_ip: 127.0.0.1
  `, nil, "Check Failed! 1 errors found:\n\terror found at " +
			"root.ResourceConfig.RootManagerInternal.KubernetesRM.InternalTaskGateway: " +
			"invalid gateway_name:  must be non-empty"},

		{"k8s missing gateway_namespace", `
resource_manager:
  type: kubernetes
  max_slots_per_pod: 1
  name: a
  internal_task_gateway:
    gateway_name: test
    gateway_ip: 127.0.0.1
  `, nil, "Check Failed! 1 errors found:\n\terror found at " +
			"root.ResourceConfig.RootManagerInternal.KubernetesRM.InternalTaskGateway: " +
			"invalid gateway_namespace:  must be non-empty"},

		{"k8s missing gateway_ip", `
resource_manager:
  type: kubernetes
  max_slots_per_pod: 1
  name: a
  internal_task_gateway:
    gateway_name: test
    gateway_namespace: abc
  `, nil, "Check Failed! 1 errors found:\n\terror found at " +
			"root.ResourceConfig.RootManagerInternal.KubernetesRM.InternalTaskGateway: " +
			"invalid gateway_ip:  must be non-empty"},
	}

	RegisterAuthZType("basic")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			unmarshaled := DefaultConfig()
			err := yaml.Unmarshal([]byte(c.yaml), &unmarshaled, yaml.DisallowUnknownFields)
			require.NoError(t, err)

			require.Equal(t, c.expectedError, unmarshaled.Resolve())
			if c.expectedError != nil {
				return
			}

			err = check.Validate(unmarshaled)
			require.Error(t, err, "expected validate to return error")
			require.Equal(t, c.validationErrorString, err.Error())
		})
	}
}

func TestResolveConfig(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		expected Config
	}{
		{"no resource manager or pools specified", `{}`, Config{
			ResourceConfig: ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{
						ClusterName:                DefaultClusterName,
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler:                  DefaultSchedulerConfig(),
					},
				},
				RootPoolsInternal: []ResourcePoolConfig{
					{
						PoolName:                 "default",
						MaxAuxContainersPerAgent: 100,
						MaxCPUContainersPerAgent: -1,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
			},
		}},

		{"old resource manager specified with no pools / no scheduler", `
resource_manager:
  type: agent`, Config{
			ResourceConfig: ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{
						ClusterName:                DefaultClusterName,
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler:                  DefaultSchedulerConfig(),
					},
				},
				RootPoolsInternal: []ResourcePoolConfig{
					{
						PoolName:                 "default",
						MaxAuxContainersPerAgent: 100,
						MaxCPUContainersPerAgent: -1,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
			},
		}},

		{"old resource manager specified with no pools / scheduler given", `
resource_manager:
  type: agent
  scheduler:
    type: fair_share`, Config{
			ResourceConfig: ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{
						ClusterName:                DefaultClusterName,
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler: &SchedulerConfig{
							FairShare:     &FairShareSchedulerConfig{},
							FittingPolicy: "best",
						},
					},
				},
				RootPoolsInternal: []ResourcePoolConfig{
					{
						PoolName:                 "default",
						MaxAuxContainersPerAgent: 100,
						MaxCPUContainersPerAgent: -1,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
			},
		}},

		{"old resource manager specified with pools given / scheduler given", `
resource_manager:
  type: agent
  scheduler:
    type: fair_share
resource_pools:
  - pool_name: test
  - pool_name: test2`, Config{
			ResourceConfig: ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{
						ClusterName:                DefaultClusterName,
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler: &SchedulerConfig{
							FairShare:     &FairShareSchedulerConfig{},
							FittingPolicy: "best",
						},
					},
				},
				RootPoolsInternal: []ResourcePoolConfig{
					{
						PoolName:                 "test",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
					{
						PoolName:                 "test2",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
			},
		}},

		{"two resource managers", `
resource_manager:
  type: agent
  name: default
  cluster_name: c1
  metadata:
    region: "nw"
resource_pools:
 - pool_name: a
 - pool_name: b
additional_resource_managers:
 - resource_manager:
     name: test
     cluster_name: c2
     type: kubernetes
     metadata:
       test: "y"
       name: k8s
     max_slots_per_pod: 65
   resource_pools:
    - pool_name: c
    - pool_name: d`, Config{
			ResourceConfig: ResourceConfig{
				RootManagerInternal: &ResourceManagerConfig{
					AgentRM: &AgentResourceManagerConfig{
						Name:                       DefaultClusterName,
						ClusterName:                "c1",
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler:                  DefaultSchedulerConfig(),
						Metadata: map[string]string{
							"region": "nw",
						},
					},
				},
				RootPoolsInternal: []ResourcePoolConfig{
					{
						PoolName:                 "a",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
					{
						PoolName:                 "b",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
				AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
					{
						ResourceManager: &ResourceManagerConfig{
							KubernetesRM: &KubernetesResourceManagerConfig{
								Name:                       "test",
								ClusterName:                "c2",
								SlotType:                   "cuda",
								DefaultAuxResourcePool:     "default",
								MaxSlotsPerPod:             ptrs.Ptr(65),
								DefaultComputeResourcePool: "default",
								Metadata: map[string]string{
									"test": "y",
									"name": "k8s",
								},
							},
						},
						ResourcePools: []ResourcePoolConfig{
							{
								PoolName:                 "c",
								MaxAuxContainersPerAgent: 100,
								AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
							},
							{
								PoolName:                 "d",
								MaxAuxContainersPerAgent: 100,
								AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
							},
						},
					},
				},
			},
		}},
	}

	RegisterAuthZType("basic")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			unmarshaled := DefaultConfig()
			err := yaml.Unmarshal([]byte(c.yaml), &unmarshaled, yaml.DisallowUnknownFields)
			require.NoError(t, err)
			require.NoError(t, unmarshaled.Resolve())
			require.NoError(t, check.Validate(unmarshaled))

			require.Equal(t, c.expected.ResourceConfig, unmarshaled.ResourceConfig)
		})
	}
}
