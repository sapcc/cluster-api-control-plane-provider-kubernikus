// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package kubernikus

import (
	"github.com/go-logr/logr"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"

	"github.com/sapcc/cluster-api-control-plane-provider-kubernikus/api/v1alpha1"
)

func (c *Client) GetKKSKubeconfig(cp *v1alpha1.KubernikusControlPlane, logger logr.Logger) (string, error) {
	logger.Info("getting kubeconfig from kubernikus")
	gccp := operations.NewGetClusterCredentialsParams()
	gccp.Name = cp.Name
	gcco, err := c.kks.Operations.GetClusterCredentials(gccp, c)
	if err != nil {
		logger.Error(err, "failed to get kubeconfig")
		return "", err
	}
	return gcco.Payload.Kubeconfig, nil
}
