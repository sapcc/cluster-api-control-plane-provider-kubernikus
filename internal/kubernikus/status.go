// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package kubernikus

import (
	"github.com/go-logr/logr"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"

	"github.com/sapcc/cluster-api-control-plane-provider-kubernikus/api/v1alpha1"
)

func (c *Client) GetKKSStatus(cp *v1alpha1.KubernikusControlPlane, logger logr.Logger) (*v1alpha1.KubernikusControlPlaneStatus, error) {
	ret := &v1alpha1.KubernikusControlPlaneStatus{}
	lcp := operations.NewListClustersParams()
	lco, err := c.kks.Operations.ListClusters(lcp, c)
	if err != nil {
		logger.Error(err, "failed to list clusters")
		return nil, err
	}
	for _, kluster := range lco.Payload {
		if kluster.Name == cp.Name {
			ret.Version = "v" + kluster.Status.ApiserverVersion
			ret.Initialized = true
			if kluster.Status.Phase == models.KlusterPhaseRunning {
				ret.Ready = true
			} else {
				ret.Ready = false
			}
			break
		}
	}

	return ret, nil
}
