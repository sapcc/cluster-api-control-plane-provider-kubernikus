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
	//kksName := buildKKSName(cp.Name, c.ConnectorId)
	lcp := operations.NewListClustersParams()
	lco, err := c.kks.Operations.ListClusters(lcp, c)
	if err != nil {
		logger.Error(err, "failed to get cluster")
		return err
	}
	for _, kluster := range lco.Payload {
		if kluster.Name == cp.Name {
			logger.Info("cluster already exists")
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

func buildKKSName(name string, conn string) string {
	return name + "-" + conn
}

func buildKlusterFromControlPlane(cp *v1alpha1.KubernikusControlPlane) *models.Kluster {
	f := false
	return &models.Kluster{
		Name: cp.Name,
		Spec: models.KlusterSpec{
			NoCloud:   true,
			Version:   cp.Spec.Version,
			Backup:    "off",
			Dashboard: &f,
			Dex:       &f,
		},
	}
}
