/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"github.com/sapcc/cluster-api-control-plane-provider-kubernikus/internal/kubernikus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/util"
	certs2 "sigs.k8s.io/cluster-api/util/certs"
	"sigs.k8s.io/cluster-api/util/kubeconfig"
	"sigs.k8s.io/cluster-api/util/secret"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"

	controlplanev1alpha1 "github.com/sapcc/cluster-api-control-plane-provider-kubernikus/api/v1alpha1"
)

// KubernikusControlPlaneReconciler reconciles a KubernikusControlPlane object
type KubernikusControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubernikuscontrolplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubernikuscontrolplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=kubernikuscontrolplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KubernikusControlPlane object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *KubernikusControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("kubernikuscontrolplane", req.NamespacedName)

	logger.Info("Reconciling KubernikusControlPlane")

	var kcp controlplanev1alpha1.KubernikusControlPlane

	err := r.Client.Get(ctx, req.NamespacedName, &kcp)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("KubernikusControlPlane may be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get KubernikusControlPlane")
		return ctrl.Result{}, err
	}

	if len(kcp.GetOwnerReferences()) == 0 {
		logger.Info("KubernikusControlPlane has no owner reference, skipping")
		return ctrl.Result{Requeue: true}, nil
	}

	// TODO: add finalizer

	cluster, err := util.GetOwnerCluster(ctx, r.Client, kcp.ObjectMeta)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "Failed to get owner cluster, stopping reconciliation")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get owner cluster")
		return ctrl.Result{}, err
	}

	logger.Info("Found owner cluster", "cluster", cluster.Name)

	// check owner cluster
	// cluster.Status.InfrastructureReady

	var sec v1.Secret
	err = r.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}, &sec)
	if err != nil {
		logger.Error(err, "Failed to get secret")
		return ctrl.Result{}, err
	}
	conv := convertSecret(&sec)
	logger.Info("Got secret", "host", conv["host"], "user", conv["user"], "conn", conv["conn"])

	kks := kubernikus.NewClient(conv["host"], conv["user"], conv["pass"], conv["conn"], conv["auth"]+"/auth/login")
	err = kks.EnsureControlPlane(&kcp, logger)
	if err != nil {
		logger.Error(err, "Failed to ensure control plane")
		return ctrl.Result{}, err
	}

	// get the latest status from kubernikus
	status, err := kks.GetKKSStatus(&kcp, logger)
	if err != nil {
		logger.Error(err, "Failed to get status")
		return ctrl.Result{}, err
	}
	// update the status of the kcp
	kcp.Status = *status
	err = r.Status().Update(ctx, &kcp)
	if err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	// set owner cp endpoint if status is ready
	if status.Ready && cluster.Spec.ControlPlaneEndpoint.Host == "" {
		ep, err := kks.GetKKSEndpoint(&kcp)
		if err != nil {
			logger.Error(err, "Failed to get endpoint")
			return ctrl.Result{}, err
		}
		cluster.Spec.ControlPlaneEndpoint = *ep
		err = r.Update(ctx, cluster)
		if err != nil {
			logger.Error(err, "Failed to update cluster")
			return ctrl.Result{}, err
		}
	}
	// set necessary secrets and labels according to status
	if status.Ready {
		// check if secret is already present
		kcSecret, err := secret.Get(ctx, r.Client, util.ObjectKey(cluster), secret.Kubeconfig)
		if err != nil {
			if errors.IsNotFound(err) {
				// if not create it
				logger.Info("Kubeconfig secret not found, creating")
				kcStr, err := kks.GetKKSKubeconfig(&kcp, logger)
				if err != nil {
					logger.Error(err, "Failed to get kubeconfig")
					return ctrl.Result{}, err
				}
				logger.Info("generating kubeconfig secret")
				kcSecret = kubeconfig.GenerateSecret(cluster, []byte(kcStr))
				err = r.Create(ctx, kcSecret)
				if err != nil {
					logger.Error(err, "Failed to create kubeconfig secret")
					return ctrl.Result{}, err
				}
				logger.Info("loading kubeconfig")
				authInfo, err := clientcmd.Load([]byte(kcStr))
				if err != nil {
					logger.Error(err, "Failed to load kubeconfig")
					return ctrl.Result{}, err
				}
				logger.Info("getting ca secret")
				caSec, err := kks.GetKKSCa(&kcp, logger)
				if err != nil {
					logger.Error(err, "Failed to get ca secret")
					return ctrl.Result{}, err
				}
				logger.Info("context", "current", authInfo.Contexts[authInfo.CurrentContext])
				aIStr := authInfo.Contexts[authInfo.CurrentContext].AuthInfo
				cCStr := authInfo.Contexts[authInfo.CurrentContext].Cluster
				saKeyData := authInfo.AuthInfos[aIStr].ClientKeyData
				saCertData := authInfo.AuthInfos[aIStr].ClientCertificateData
				saCert := secret.Certificate{
					Purpose: secret.ServiceAccount,
					KeyPair: &certs2.KeyPair{
						Cert: saCertData,
						Key:  saKeyData,
					},
					External:  true,
					Generated: true,
				}
				caCert := secret.Certificate{
					Purpose: secret.ClusterCA,
					KeyPair: &certs2.KeyPair{
						Key:  []byte(caSec.StringData["tls.key"]),
						Cert: authInfo.Clusters[cCStr].CertificateAuthorityData,
					},
					External:  true,
					Generated: true,
				}
				certs := secret.Certificates{&saCert, &caCert}
				gvk := controlplanev1alpha1.GroupVersion.WithKind("KubernikusControlPlane")
				f := false
				cRef := metav1.OwnerReference{
					APIVersion:         controlplanev1alpha1.GroupVersion.String(),
					Kind:               gvk.Kind,
					Name:               kcp.Name,
					UID:                kcp.UID,
					Controller:         &f,
					BlockOwnerDeletion: &f,
				}
				logger.Info("presavegen", "object", util.ObjectKey(cluster), "cRef", cRef)
				err = certs.SaveGenerated(ctx, r.Client, util.ObjectKey(cluster), cRef)
				if err != nil {
					logger.Error(err, "Failed to create secrets")
					return ctrl.Result{}, err
				}
			} else {
				logger.Error(err, "Failed to get kubeconfig secret")
				return ctrl.Result{}, err
			}
		}
		// if yes - check if it needs rotation
		rotate, err := kubeconfig.NeedsClientCertRotation(kcSecret, time.Minute*30)
		if err != nil {
			logger.Error(err, "Failed to check kubeconfig for rotation")
			return ctrl.Result{}, err
		}
		if rotate {
			logger.Info("Kubeconfig needs rotation, updating")
			kcStr, err := kks.GetKKSKubeconfig(&kcp, logger)
			if err != nil {
				logger.Error(err, "Failed to get kubeconfig")
				return ctrl.Result{}, err
			}
			kcSecret = kubeconfig.GenerateSecret(cluster, []byte(kcStr))
			err = r.Update(ctx, kcSecret)
			if err != nil {
				logger.Error(err, "Failed to update kubeconfig secret")
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubernikusControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&controlplanev1alpha1.KubernikusControlPlane{}).
		Complete(r)
}

// convertSecret takes a v1.Secret and converts it to a string map for use in the kubernikus client
// TODO: revisit this and create a proper struct
func convertSecret(sec *v1.Secret) map[string]string {
	ret := make(map[string]string)
	for k, v := range sec.Data {
		ret[k] = strings.TrimSuffix(string(v), "\n")
	}
	return ret
}
