// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package kubernikus

import (
	"net/url"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/sapcc/cluster-api-control-plane-provider-kubernikus/api/v1alpha1"
)

func (c *Client) GetKKSEndpoint(cp *v1alpha1.KubernikusControlPlane) (*v1beta1.APIEndpoint, error) {
	ret := v1beta1.APIEndpoint{}
	scp := operations.ShowClusterParams{Name: cp.Name}
	sco, err := c.kks.Operations.ShowCluster(&scp, c)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(sco.Payload.Status.Apiserver)
	if err != nil {
		return nil, err
	}
	ret.Host = u.Hostname()
	// TODO: revisit this is too hardcoded
	ret.Port = 443

	return &ret, nil
}
