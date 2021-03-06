package controllers

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	upgrademgrv1alpha1 "github.com/keikoproj/upgrade-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestGetMaxUnavailableWithPercentageValue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	strategy := upgrademgrv1alpha1.UpdateStrategy{
		MaxUnavailable: intstr.Parse("75%"),
	}
	g.Expect(getMaxUnavailable(strategy, 200)).To(gomega.Equal(150))
}

func TestIsNodeReady(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	tt := map[corev1.NodeCondition]bool{
		corev1.NodeCondition{Type: corev1.NodeReady, Status: corev1.ConditionTrue}:  true,
		corev1.NodeCondition{Type: corev1.NodeReady, Status: corev1.ConditionFalse}: false,
	}

	for condition, val := range tt {
		node := corev1.Node{
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					condition,
				},
			},
		}
		g.Expect(isNodeReady(node)).To(gomega.Equal(val))
	}
}

func TestIsNodePassesReadinessGates(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type test struct {
		gate   []map[string]string
		labels map[string]string
		want   bool
	}
	tests := []test{
		{
			gate: []map[string]string{
				{
					"healthy": "true",
				},
			},
			labels: map[string]string{
				"healthy": "true",
			},
			want: true,
		},

		{
			gate: []map[string]string{},
			labels: map[string]string{
				"healthy": "true",
			},
			want: true,
		},

		{
			gate: []map[string]string{
				{"healthy": "true"},
			},
			labels: map[string]string{
				"healthy": "false",
			},
			want: false,
		},

		{
			gate: []map[string]string{
				{"healthy": "true"},
			},
			labels: map[string]string{},
			want:   false,
		},

		{
			gate: []map[string]string{
				{"healthy": "true"},
				{"second-check": "true"},
			},
			labels: map[string]string{
				"healthy": "true",
			},
			want: false,
		},
		{
			gate: []map[string]string{
				{"healthy": "true"},
				{"second-check": "true"},
			},
			labels: map[string]string{
				"healthy":      "true",
				"second-check": "true",
			},
			want: true,
		},
		{
			gate: []map[string]string{
				{"healthy": "true"},
				{"second-check": "true"},
			},
			labels: map[string]string{
				"healthy":      "true",
				"second-check": "false",
			},
			want: false,
		}}

	for _, tt := range tests {
		readinessGates := make([]upgrademgrv1alpha1.NodeReadinessGate, len(tt.gate))
		for i, g := range tt.gate {
			readinessGates[i] = upgrademgrv1alpha1.NodeReadinessGate{
				MatchLabels: g,
			}
		}
		node := corev1.Node{
			ObjectMeta: v1.ObjectMeta{
				Labels: tt.labels,
			},
		}
		g.Expect(isNodePassingReadinessGates(node, readinessGates)).To(gomega.Equal(tt.want))
	}

}

func TestGetInServiceCount(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	tt := map[*autoscaling.Instance]int64{
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateInService)}:          1,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateDetached)}:           0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateDetaching)}:          0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateEnteringStandby)}:    0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStatePending)}:            0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStatePendingProceed)}:     0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStatePendingWait)}:        0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateQuarantined)}:        0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateStandby)}:            0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateTerminated)}:         0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateTerminating)}:        0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateTerminatingProceed)}: 0,
		&autoscaling.Instance{LifecycleState: aws.String(autoscaling.LifecycleStateTerminatingWait)}:    0,
	}

	// test every condition
	for instance, expectedCount := range tt {
		instances := []*autoscaling.Instance{
			instance,
		}
		g.Expect(getInServiceCount(instances)).To(gomega.Equal(expectedCount))
	}

	// test all instances
	instances := []*autoscaling.Instance{}
	for instance := range tt {
		instances = append(instances, instance)
	}
	g.Expect(getInServiceCount(instances)).To(gomega.Equal(int64(1)))
}

func TestGetInServiceIds(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	tt := map[*autoscaling.Instance][]string{
		&autoscaling.Instance{InstanceId: aws.String("i-1"), LifecycleState: aws.String(autoscaling.LifecycleStateInService)}:           {"i-1"},
		&autoscaling.Instance{InstanceId: aws.String("i-2"), LifecycleState: aws.String(autoscaling.LifecycleStateDetached)}:            {},
		&autoscaling.Instance{InstanceId: aws.String("i-3"), LifecycleState: aws.String(autoscaling.LifecycleStateDetaching)}:           {},
		&autoscaling.Instance{InstanceId: aws.String("i-4"), LifecycleState: aws.String(autoscaling.LifecycleStateEnteringStandby)}:     {},
		&autoscaling.Instance{InstanceId: aws.String("i-5"), LifecycleState: aws.String(autoscaling.LifecycleStatePending)}:             {},
		&autoscaling.Instance{InstanceId: aws.String("i-6"), LifecycleState: aws.String(autoscaling.LifecycleStatePendingProceed)}:      {},
		&autoscaling.Instance{InstanceId: aws.String("i-7"), LifecycleState: aws.String(autoscaling.LifecycleStatePendingWait)}:         {},
		&autoscaling.Instance{InstanceId: aws.String("i-8"), LifecycleState: aws.String(autoscaling.LifecycleStateQuarantined)}:         {},
		&autoscaling.Instance{InstanceId: aws.String("i-9"), LifecycleState: aws.String(autoscaling.LifecycleStateStandby)}:             {},
		&autoscaling.Instance{InstanceId: aws.String("i-10"), LifecycleState: aws.String(autoscaling.LifecycleStateTerminated)}:         {},
		&autoscaling.Instance{InstanceId: aws.String("i-11"), LifecycleState: aws.String(autoscaling.LifecycleStateTerminating)}:        {},
		&autoscaling.Instance{InstanceId: aws.String("i-12"), LifecycleState: aws.String(autoscaling.LifecycleStateTerminatingProceed)}: {},
		&autoscaling.Instance{InstanceId: aws.String("i-13"), LifecycleState: aws.String(autoscaling.LifecycleStateTerminatingWait)}:    {},
	}

	// test every condition
	for instance, expectedList := range tt {
		instances := []*autoscaling.Instance{
			instance,
		}
		g.Expect(getInServiceIds(instances)).To(gomega.Equal(expectedList))
	}

	// test all instances
	instances := []*autoscaling.Instance{}
	for instance := range tt {
		instances = append(instances, instance)
	}
	g.Expect(getInServiceIds(instances)).To(gomega.Equal([]string{"i-1"}))
}

func TestGetMaxUnavailableWithPercentageValue33(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	strategy := upgrademgrv1alpha1.UpdateStrategy{
		MaxUnavailable: intstr.Parse("67%"),
	}
	g.Expect(getMaxUnavailable(strategy, 3)).To(gomega.Equal(2))
}

func TestGetMaxUnavailableWithPercentageAndSingleInstance(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	totalNodes := 1
	strategy := upgrademgrv1alpha1.UpdateStrategy{
		MaxUnavailable: intstr.Parse("67%"),
	}
	g.Expect(getMaxUnavailable(strategy, totalNodes)).To(gomega.Equal(1))
}

func TestGetMaxUnavailableWithPercentageNonIntResult(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	strategy := upgrademgrv1alpha1.UpdateStrategy{
		MaxUnavailable: intstr.Parse("37%"),
	}
	g.Expect(getMaxUnavailable(strategy, 50)).To(gomega.Equal(18))
}

func TestGetMaxUnavailableWithIntValue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	strategy := upgrademgrv1alpha1.UpdateStrategy{
		MaxUnavailable: intstr.Parse("75"),
	}
	g.Expect(getMaxUnavailable(strategy, 200)).To(gomega.Equal(75))
}

func TestGetNextAvailableInstance(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mockAsgName := "some-asg"
	mockInstanceName1 := "foo1"
	mockInstanceName2 := "bar1"
	az := "az-1"
	instance1 := autoscaling.Instance{InstanceId: &mockInstanceName1, AvailabilityZone: &az}
	instance2 := autoscaling.Instance{InstanceId: &mockInstanceName2, AvailabilityZone: &az}

	instancesList := []*autoscaling.Instance{&instance1, &instance2}
	rcRollingUpgrade := &RollingUpgradeReconciler{ClusterState: clusterState}
	rcRollingUpgrade.ClusterState.initializeAsg(mockAsgName, instancesList)
	available := getNextAvailableInstances(mockAsgName, 1, instancesList, rcRollingUpgrade.ClusterState)

	g.Expect(1).Should(gomega.Equal(len(available)))
	g.Expect(rcRollingUpgrade.ClusterState.deleteAllInstancesInAsg(mockAsgName)).To(gomega.BeTrue())

}

func TestGetNextAvailableInstanceNoInstanceFound(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mockAsgName := "some-asg"
	mockInstanceName1 := "foo1"
	mockInstanceName2 := "bar1"
	az := "az-1"
	instance1 := autoscaling.Instance{InstanceId: &mockInstanceName1, AvailabilityZone: &az}
	instance2 := autoscaling.Instance{InstanceId: &mockInstanceName2, AvailabilityZone: &az}

	instancesList := []*autoscaling.Instance{&instance1, &instance2}
	rcRollingUpgrade := &RollingUpgradeReconciler{ClusterState: clusterState}
	rcRollingUpgrade.ClusterState.initializeAsg(mockAsgName, instancesList)
	available := getNextAvailableInstances("asg2", 1, instancesList, rcRollingUpgrade.ClusterState)

	g.Expect(0).Should(gomega.Equal(len(available)))
	g.Expect(rcRollingUpgrade.ClusterState.deleteAllInstancesInAsg(mockAsgName)).To(gomega.BeTrue())

}

func TestGetNextAvailableInstanceInAz(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mockAsgName := "some-asg"
	mockInstanceName1 := "foo1"
	mockInstanceName2 := "bar1"
	az := "az-1"
	az2 := "az-2"
	instance1 := autoscaling.Instance{InstanceId: &mockInstanceName1, AvailabilityZone: &az}
	instance2 := autoscaling.Instance{InstanceId: &mockInstanceName2, AvailabilityZone: &az2}

	instancesList := []*autoscaling.Instance{&instance1, &instance2}
	rcRollingUpgrade := &RollingUpgradeReconciler{ClusterState: clusterState}
	rcRollingUpgrade.ClusterState.initializeAsg(mockAsgName, instancesList)

	instances := getNextSetOfAvailableInstancesInAz(mockAsgName, az, 1, instancesList, rcRollingUpgrade.ClusterState)
	g.Expect(1).Should(gomega.Equal(len(instances)))
	g.Expect(mockInstanceName1).Should(gomega.Equal(*instances[0].InstanceId))

	instances = getNextSetOfAvailableInstancesInAz(mockAsgName, az2, 1, instancesList, rcRollingUpgrade.ClusterState)
	g.Expect(1).Should(gomega.Equal(len(instances)))
	g.Expect(mockInstanceName2).Should(gomega.Equal(*instances[0].InstanceId))

	instances = getNextSetOfAvailableInstancesInAz(mockAsgName, "az3", 1, instancesList, rcRollingUpgrade.ClusterState)
	g.Expect(0).Should(gomega.Equal(len(instances)))

	g.Expect(rcRollingUpgrade.ClusterState.deleteAllInstancesInAsg(mockAsgName)).To(gomega.BeTrue())

}

func TestGetNextAvailableInstanceInAzGetMultipleInstances(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mockAsgName := "some-asg"
	mockInstanceName1 := "foo1"
	mockInstanceName2 := "bar1"
	az := "az-1"
	instance1 := autoscaling.Instance{InstanceId: &mockInstanceName1, AvailabilityZone: &az}
	instance2 := autoscaling.Instance{InstanceId: &mockInstanceName2, AvailabilityZone: &az}

	instancesList := []*autoscaling.Instance{&instance1, &instance2}
	rcRollingUpgrade := &RollingUpgradeReconciler{ClusterState: clusterState}
	rcRollingUpgrade.ClusterState.initializeAsg(mockAsgName, instancesList)

	instances := getNextSetOfAvailableInstancesInAz(mockAsgName, az, 3, instancesList, rcRollingUpgrade.ClusterState)

	// Even though the request is for 3 instances, only 2 should be returned as there are only 2 nodes in the ASG
	g.Expect(2).Should(gomega.Equal(len(instances)))
	instanceIds := []string{*instances[0].InstanceId, *instances[1].InstanceId}
	g.Expect(instanceIds).Should(gomega.ConsistOf(mockInstanceName1, mockInstanceName2))

	g.Expect(rcRollingUpgrade.ClusterState.deleteAllInstancesInAsg(mockAsgName)).To(gomega.BeTrue())
}
