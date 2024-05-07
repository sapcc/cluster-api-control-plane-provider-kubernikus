package kubernikus

import (
	"github.com/go-logr/logr"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/sapcc/cluster-api-control-plane-provider-kubernikus/api/v1alpha1"
	kksClient "github.com/sapcc/kubernikus/pkg/api/client"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"time"
)

type Client struct {
	Host        string
	Token       string
	Username    string
	Password    string
	ConnectorId string
	AuthUrl     string
	TokenTime   time.Time
	kks         *kksClient.Kubernikus
}

func NewClient(host, username, password, connectorId, authUrl string) *Client {
	return &Client{
		Host:        host,
		Username:    username,
		Password:    password,
		ConnectorId: connectorId,
		AuthUrl:     authUrl,
		kks:         kksClient.NewHTTPClientWithConfig(nil, kksClient.DefaultTransportConfig().WithHost(host)),
	}
}

func (c *Client) AuthenticateRequest(req runtime.ClientRequest, reg strfmt.Registry) error {
	var err error
	if time.Since(c.TokenTime) > 30*time.Minute {
		c.Token, err = GetToken(c.Username, c.Password, c.ConnectorId, c.AuthUrl)
		if err != nil {
			return err
		}
		c.TokenTime = time.Now()
	}

	err = req.SetHeaderParam("Authorization", "Bearer "+c.Token)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) EnsureControlPlane(cp *v1alpha1.KubernikusControlPlane, logger logr.Logger) error {
	lcp := operations.NewListClustersParams()
	lco, err := c.kks.Operations.ListClusters(lcp, c)
	if err != nil {
		logger.Error(err, "failed to get cluster")
		return err
	}
	for _, kluster := range lco.Payload {
		if kluster.Name == cp.Name {
			logger.Info("cluster already exists")
			scp := operations.NewShowClusterParams()
			scp.Name = cp.Name
			sco, err := c.kks.Operations.ShowCluster(scp, c)
			if err != nil {
				logger.Error(err, "failed to get cluster")
				return err
			}
			// this only updates the kks kluster if the version changes
			// TODO: revisit this
			if sco.Payload.Spec.Version != cp.Spec.Version {
				logger.Info("cluster version does not match, updating")
				ucp := operations.NewUpdateClusterParams()
				ucp.Name = cp.Name
				ucp.Body = buildKlusterFromControlPlane(cp)
				_, err := c.kks.Operations.UpdateCluster(ucp, c)
				if err != nil {
					logger.Error(err, "failed to update cluster")
					return err
				}
			}
			return nil
		}
	}
	logger.Info("cluster does not exist, creating")
	ncp := operations.NewCreateClusterParams()
	ncp.Body = buildKlusterFromControlPlane(cp)
	ncco, err := c.kks.Operations.CreateCluster(ncp, c)
	if err != nil {
		logger.Error(err, "failed to create cluster")
		return err
	}
	logger.Info("cluster created", "name", ncco.Payload.Name)
	return nil
}

func buildKlusterFromControlPlane(cp *v1alpha1.KubernikusControlPlane) *models.Kluster {
	f := false
	audit := "stdout"
	ret := &models.Kluster{
		Name: cp.Name,
		Spec: models.KlusterSpec{
			NoCloud:     true,
			Version:     cp.Spec.Version,
			CustomCNI:   true,
			SeedKubeadm: true,
			Dashboard:   &f,
			Dex:         &f,
			Audit:       &audit,
		},
	}

	if cp.Spec.ServiceCidr != "" {
		ret.Spec.ServiceCIDR = cp.Spec.ServiceCidr
	}
	if cp.Spec.ClusterCidr != "" {
		ret.Spec.ClusterCIDR = &cp.Spec.ClusterCidr
	}
	if cp.Spec.AdvertisePort != 0 {
		ret.Spec.AdvertisePort = cp.Spec.AdvertisePort
	}
	if cp.Spec.AdvertiseAddress != "" {
		ret.Spec.AdvertiseAddress = cp.Spec.AdvertiseAddress
	}
	if cp.Spec.Backup != "" {
		ret.Spec.Backup = cp.Spec.Backup
	}
	if cp.Spec.DnsDomain != "" {
		ret.Spec.DNSDomain = cp.Spec.DnsDomain
	}
	if cp.Spec.DnsAddress != "" {
		ret.Spec.DNSAddress = cp.Spec.DnsAddress
	}
	if cp.Spec.SSHPublicKey != "" {
		ret.Spec.SSHPublicKey = cp.Spec.SSHPublicKey
	}
	if cp.Spec.Oidc != nil {
		ret.Spec.Oidc = &models.OIDC{
			IssuerURL: cp.Spec.Oidc.IssuerURL,
			ClientID:  cp.Spec.Oidc.ClientID,
		}
	}

	return ret
}
