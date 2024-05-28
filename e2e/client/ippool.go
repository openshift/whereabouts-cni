// This code is based on code from the following repository
// https://github.com/bcreane/k8sutils

package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	kubeClient "github.com/k8snetworkplumbingwg/whereabouts/pkg/storage/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func isIPPoolAllocationsEmpty(ctx context.Context, k8sIPAM *kubeClient.KubernetesIPAM, ipPoolCIDR string) wait.ConditionWithContextFunc {
	return func(context.Context) (bool, error) {
		ipPool, err := k8sIPAM.GetIPPool(ctx, kubeClient.PoolIdentifier{IpRange: ipPoolCIDR, NetworkName: kubeClient.UnnamedNetwork})
		noPoolError := fmt.Errorf("k8s pool initialized")
		if errors.Is(err, noPoolError) {
			return true, nil
		} else if err != nil {
			return false, err
		}

		if len(ipPool.Allocations()) != 0 {
			return false, nil
		}

		return true, nil
	}
}

func isIPPoolAllocationsEmptyForNodeSlices(ctx context.Context, k8sIPAM *kubeClient.KubernetesIPAM, ipPoolCIDR string, clientInfo *ClientInfo) wait.ConditionWithContextFunc {
	return func(context.Context) (bool, error) {
		nodes, err := clientInfo.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, node := range nodes.Items {
			ipPool, err := k8sIPAM.GetIPPool(ctx, kubeClient.PoolIdentifier{NodeName: node.Name, IpRange: ipPoolCIDR, NetworkName: k8sIPAM.Config.NetworkName})
			if err != nil {
				if err.Error() == "k8s pool initialized" {
					continue
				} else {
					return false, err
				}
			}

			if len(ipPool.Allocations()) != 0 {
				return false, nil
			}
		}
		return true, nil
	}
}

// WaitForZeroIPPoolAllocations polls up to timeout seconds for IP pool allocations to be gone from the Kubernetes cluster.
// Returns an error if any IP pool allocations remain after time limit, or if GETing IP pools causes an error.
func WaitForZeroIPPoolAllocations(ctx context.Context, k8sIPAM *kubeClient.KubernetesIPAM, ipPoolCIDR string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, isIPPoolAllocationsEmpty(ctx, k8sIPAM, ipPoolCIDR))
}

// WaitForZeroIPPoolAllocationsAcrossNodeSlices polls up to timeout seconds for IP pool allocations to be gone from the Kubernetes cluster.
// Returns an error if any IP pool allocations remain after time limit, or if GETing IP pools causes an error.
func WaitForZeroIPPoolAllocationsAcrossNodeSlices(ctx context.Context, k8sIPAM *kubeClient.KubernetesIPAM, ipPoolCIDR string, timeout time.Duration, clientInfo *ClientInfo) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, isIPPoolAllocationsEmptyForNodeSlices(ctx, k8sIPAM, ipPoolCIDR, clientInfo))
}
