// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package kubernikus

import (
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	corev1 "k8s.io/api/core/v1"

	"github.com/sapcc/cluster-api-control-plane-provider-kubernikus/api/v1alpha1"
)

func (c *Client) GetKKSCa(cp *v1alpha1.KubernikusControlPlane, logger logr.Logger) (corev1.Secret, error) {
	logger.Info("getting ca secret from kubernikus")
	gccp := operations.NewGetClusterKubeadmSecretParams()
	gccp.Name = cp.Name
	gcco, err := c.kks.Operations.GetClusterKubeadmSecret(gccp, c)
	if err != nil {
		logger.Error(err, "failed to get ca secret")
		return corev1.Secret{}, err
	}
	kadmSecret := corev1.Secret{}
	err = yaml.Unmarshal([]byte(gcco.Payload.Secret), &kadmSecret)
	if err != nil {
		logger.Error(err, "failed to unmarshal ca secret")
		return corev1.Secret{}, err
	}
	return kadmSecret, nil
}
