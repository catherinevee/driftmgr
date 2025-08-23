package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubernetesProvider implements CloudProvider for Kubernetes clusters
type KubernetesProvider struct {
	client      *kubernetes.Clientset
	config      *rest.Config
	namespace   string
	clusterName string
}

// NewKubernetesProvider creates a new Kubernetes provider
func NewKubernetesProvider() (*KubernetesProvider, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get cluster name from context or environment
	clusterName := os.Getenv("KUBE_CLUSTER_NAME")
	if clusterName == "" {
		clusterName = "kubernetes-cluster"
	}

	return &KubernetesProvider{
		client:      clientset,
		config:      config,
		namespace:   "", // Empty means all namespaces
		clusterName: clusterName,
	}, nil
}

// getKubeConfig gets kubernetes config from various sources
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first (for when running inside k8s)
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Try kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	if kubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			return config, nil
		}
	}

	return nil, fmt.Errorf("no kubernetes configuration found")
}

// DiscoverResources discovers all Kubernetes resources
func (p *KubernetesProvider) DiscoverResources(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover Deployments
	deployments, err := p.discoverDeployments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover deployments: %w", err)
	}
	resources = append(resources, deployments...)

	// Discover Services
	services, err := p.discoverServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}
	resources = append(resources, services...)

	// Discover ConfigMaps
	configMaps, err := p.discoverConfigMaps(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover configmaps: %w", err)
	}
	resources = append(resources, configMaps...)

	// Discover Secrets
	secrets, err := p.discoverSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover secrets: %w", err)
	}
	resources = append(resources, secrets...)

	// Discover Ingresses
	ingresses, err := p.discoverIngresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover ingresses: %w", err)
	}
	resources = append(resources, ingresses...)

	// Discover Pods
	pods, err := p.discoverPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover pods: %w", err)
	}
	resources = append(resources, pods...)

	// Discover StatefulSets
	statefulSets, err := p.discoverStatefulSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover statefulsets: %w", err)
	}
	resources = append(resources, statefulSets...)

	// Discover DaemonSets
	daemonSets, err := p.discoverDaemonSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover daemonsets: %w", err)
	}
	resources = append(resources, daemonSets...)

	// Discover Jobs
	jobs, err := p.discoverJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover jobs: %w", err)
	}
	resources = append(resources, jobs...)

	// Discover CronJobs
	cronJobs, err := p.discoverCronJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover cronjobs: %w", err)
	}
	resources = append(resources, cronJobs...)

	// Discover PersistentVolumes
	pvs, err := p.discoverPersistentVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover persistent volumes: %w", err)
	}
	resources = append(resources, pvs...)

	// Discover PersistentVolumeClaims
	pvcs, err := p.discoverPersistentVolumeClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover persistent volume claims: %w", err)
	}
	resources = append(resources, pvcs...)

	// Discover StorageClasses
	storageClasses, err := p.discoverStorageClasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover storage classes: %w", err)
	}
	resources = append(resources, storageClasses...)

	// Discover NetworkPolicies
	networkPolicies, err := p.discoverNetworkPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover network policies: %w", err)
	}
	resources = append(resources, networkPolicies...)

	// Discover RBAC resources
	rbacResources, err := p.discoverRBACResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover RBAC resources: %w", err)
	}
	resources = append(resources, rbacResources...)

	// Discover HorizontalPodAutoscalers
	hpas, err := p.discoverHPAs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover HPAs: %w", err)
	}
	resources = append(resources, hpas...)

	// Discover PodDisruptionBudgets
	pdbs, err := p.discoverPDBs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover PDBs: %w", err)
	}
	resources = append(resources, pdbs...)

	return resources, nil
}

// discoverDeployments discovers all deployments
func (p *KubernetesProvider) discoverDeployments(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	deployments, err := p.client.AppsV1().Deployments(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, dep := range deployments.Items {
		resource := p.deploymentToResource(&dep)
		resources = append(resources, resource)
	}

	return resources, nil
}

// deploymentToResource converts a Kubernetes deployment to a Resource
func (p *KubernetesProvider) deploymentToResource(dep *v1.Deployment) models.Resource {
	tags := make(map[string]interface{})
	for k, v := range dep.Labels {
		tags[k] = v
	}

	state := "running"
	if dep.Status.Replicas == 0 {
		state = "stopped"
	} else if dep.Status.ReadyReplicas < dep.Status.Replicas {
		state = "degraded"
	}

	return models.Resource{
		ID:           fmt.Sprintf("%s/%s/%s", p.clusterName, dep.Namespace, dep.Name),
		Name:         dep.Name,
		Type:         "kubernetes_deployment",
		Provider:     "kubernetes",
		Region:       p.clusterName,
		State:        state,
		Tags:         tags,
		CreatedAt:    dep.CreationTimestamp.Time,
		Updated:    time.Now(),
		Attributes: map[string]interface{}{
			"namespace":       dep.Namespace,
			"replicas":        *dep.Spec.Replicas,
			"ready_replicas":  dep.Status.ReadyReplicas,
			"available":       dep.Status.AvailableReplicas,
			"strategy":        dep.Spec.Strategy.Type,
			"resource_version": dep.ResourceVersion,
		},
	}
}

// discoverServices discovers all services
func (p *KubernetesProvider) discoverServices(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	services, err := p.client.CoreV1().Services(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, svc := range services.Items {
		resource := p.serviceToResource(&svc)
		resources = append(resources, resource)
	}

	return resources, nil
}

// serviceToResource converts a Kubernetes service to a Resource
func (p *KubernetesProvider) serviceToResource(svc *corev1.Service) models.Resource {
	tags := make(map[string]interface{})
	for k, v := range svc.Labels {
		tags[k] = v
	}

	var ports []map[string]interface{}
	for _, port := range svc.Spec.Ports {
		ports = append(ports, map[string]interface{}{
			"name":        port.Name,
			"port":        port.Port,
			"target_port": port.TargetPort.String(),
			"protocol":    string(port.Protocol),
		})
	}

	return models.Resource{
		ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, svc.Namespace, svc.Name),
		Name:     svc.Name,
		Type:     "kubernetes_service",
		Provider: "kubernetes",
		Region:   p.clusterName,
		State:    "active",
		Tags:     tags,
		CreatedAt: svc.CreationTimestamp.Time,
		Updated: time.Now(),
		Attributes: map[string]interface{}{
			"namespace":    svc.Namespace,
			"type":         string(svc.Spec.Type),
			"cluster_ip":   svc.Spec.ClusterIP,
			"ports":        ports,
			"selector":     svc.Spec.Selector,
			"external_ips": svc.Spec.ExternalIPs,
		},
	}
}

// discoverConfigMaps discovers all ConfigMaps
func (p *KubernetesProvider) discoverConfigMaps(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	configMaps, err := p.client.CoreV1().ConfigMaps(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, cm := range configMaps.Items {
		tags := make(map[string]interface{})
		for k, v := range cm.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, cm.Namespace, cm.Name),
			Name:     cm.Name,
			Type:     "kubernetes_configmap",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: cm.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace": cm.Namespace,
				"data_keys": getMapKeys(cm.Data),
			},
		})
	}

	return resources, nil
}

// discoverSecrets discovers all Secrets (metadata only, not the actual secret data)
func (p *KubernetesProvider) discoverSecrets(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	secrets, err := p.client.CoreV1().Secrets(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets.Items {
		tags := make(map[string]interface{})
		for k, v := range secret.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, secret.Namespace, secret.Name),
			Name:     secret.Name,
			Type:     "kubernetes_secret",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: secret.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace": secret.Namespace,
				"type":      string(secret.Type),
				"data_keys": getSecretKeys(secret.Data),
			},
		})
	}

	return resources, nil
}

// discoverIngresses discovers all Ingresses
func (p *KubernetesProvider) discoverIngresses(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	ingresses, err := p.client.NetworkingV1().Ingresses(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ing := range ingresses.Items {
		resource := p.ingressToResource(&ing)
		resources = append(resources, resource)
	}

	return resources, nil
}

// ingressToResource converts an Ingress to a Resource
func (p *KubernetesProvider) ingressToResource(ing *networkingv1.Ingress) models.Resource {
	tags := make(map[string]interface{})
	for k, v := range ing.Labels {
		tags[k] = v
	}

	var hosts []string
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}

	return models.Resource{
		ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, ing.Namespace, ing.Name),
		Name:     ing.Name,
		Type:     "kubernetes_ingress",
		Provider: "kubernetes",
		Region:   p.clusterName,
		State:    "active",
		Tags:     tags,
		CreatedAt: ing.CreationTimestamp.Time,
		Updated: time.Now(),
		Attributes: map[string]interface{}{
			"namespace":      ing.Namespace,
			"hosts":          hosts,
			"ingress_class":  ing.Spec.IngressClassName,
			"tls_configured": len(ing.Spec.TLS) > 0,
		},
	}
}

// discoverPods discovers all Pods
func (p *KubernetesProvider) discoverPods(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pods, err := p.client.CoreV1().Pods(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		tags := make(map[string]interface{})
		for k, v := range pod.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, pod.Namespace, pod.Name),
			Name:     pod.Name,
			Type:     "kubernetes_pod",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    string(pod.Status.Phase),
			Tags:     tags,
			CreatedAt: pod.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":       pod.Namespace,
				"node_name":       pod.Spec.NodeName,
				"container_count": len(pod.Spec.Containers),
				"restart_count":   getPodRestartCount(&pod),
			},
		})
	}

	return resources, nil
}

// discoverStatefulSets discovers all StatefulSets
func (p *KubernetesProvider) discoverStatefulSets(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	statefulSets, err := p.client.AppsV1().StatefulSets(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ss := range statefulSets.Items {
		tags := make(map[string]interface{})
		for k, v := range ss.Labels {
			tags[k] = v
		}

		state := "running"
		if ss.Status.Replicas == 0 {
			state = "stopped"
		} else if ss.Status.ReadyReplicas < ss.Status.Replicas {
			state = "degraded"
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, ss.Namespace, ss.Name),
			Name:     ss.Name,
			Type:     "kubernetes_statefulset",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    state,
			Tags:     tags,
			CreatedAt: ss.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":      ss.Namespace,
				"replicas":       *ss.Spec.Replicas,
				"ready_replicas": ss.Status.ReadyReplicas,
				"service_name":   ss.Spec.ServiceName,
			},
		})
	}

	return resources, nil
}

// discoverDaemonSets discovers all DaemonSets
func (p *KubernetesProvider) discoverDaemonSets(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	daemonSets, err := p.client.AppsV1().DaemonSets(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, ds := range daemonSets.Items {
		tags := make(map[string]interface{})
		for k, v := range ds.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, ds.Namespace, ds.Name),
			Name:     ds.Name,
			Type:     "kubernetes_daemonset",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "running",
			Tags:     tags,
			CreatedAt: ds.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":          ds.Namespace,
				"desired_number":     ds.Status.DesiredNumberScheduled,
				"current_number":     ds.Status.CurrentNumberScheduled,
				"ready_number":       ds.Status.NumberReady,
				"available_number":   ds.Status.NumberAvailable,
			},
		})
	}

	return resources, nil
}

// discoverJobs discovers all Jobs
func (p *KubernetesProvider) discoverJobs(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	jobs, err := p.client.BatchV1().Jobs(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, job := range jobs.Items {
		tags := make(map[string]interface{})
		for k, v := range job.Labels {
			tags[k] = v
		}

		state := "running"
		if job.Status.CompletionTime != nil {
			state = "completed"
		} else if job.Status.Failed > 0 {
			state = "failed"
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, job.Namespace, job.Name),
			Name:     job.Name,
			Type:     "kubernetes_job",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    state,
			Tags:     tags,
			CreatedAt: job.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":  job.Namespace,
				"succeeded":  job.Status.Succeeded,
				"failed":     job.Status.Failed,
				"active":     job.Status.Active,
				"completions": job.Spec.Completions,
			},
		})
	}

	return resources, nil
}

// discoverCronJobs discovers all CronJobs
func (p *KubernetesProvider) discoverCronJobs(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	cronJobs, err := p.client.BatchV1().CronJobs(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, cj := range cronJobs.Items {
		tags := make(map[string]interface{})
		for k, v := range cj.Labels {
			tags[k] = v
		}

		state := "active"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			state = "suspended"
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, cj.Namespace, cj.Name),
			Name:     cj.Name,
			Type:     "kubernetes_cronjob",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    state,
			Tags:     tags,
			CreatedAt: cj.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace": cj.Namespace,
				"schedule":  cj.Spec.Schedule,
				"suspended": cj.Spec.Suspend != nil && *cj.Spec.Suspend,
				"active":    len(cj.Status.Active),
			},
		})
	}

	return resources, nil
}

// discoverPersistentVolumes discovers all PersistentVolumes
func (p *KubernetesProvider) discoverPersistentVolumes(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pvs, err := p.client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pv := range pvs.Items {
		tags := make(map[string]interface{})
		for k, v := range pv.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/pv/%s", p.clusterName, pv.Name),
			Name:     pv.Name,
			Type:     "kubernetes_persistent_volume",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    string(pv.Status.Phase),
			Tags:     tags,
			CreatedAt: pv.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"capacity":       pv.Spec.Capacity.Storage().String(),
				"access_modes":   pv.Spec.AccessModes,
				"reclaim_policy": string(pv.Spec.PersistentVolumeReclaimPolicy),
				"storage_class":  pv.Spec.StorageClassName,
			},
		})
	}

	return resources, nil
}

// discoverPersistentVolumeClaims discovers all PersistentVolumeClaims
func (p *KubernetesProvider) discoverPersistentVolumeClaims(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pvcs, err := p.client.CoreV1().PersistentVolumeClaims(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pvc := range pvcs.Items {
		tags := make(map[string]interface{})
		for k, v := range pvc.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, pvc.Namespace, pvc.Name),
			Name:     pvc.Name,
			Type:     "kubernetes_persistent_volume_claim",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    string(pvc.Status.Phase),
			Tags:     tags,
			CreatedAt: pvc.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":      pvc.Namespace,
				"volume_name":    pvc.Spec.VolumeName,
				"storage_class":  *pvc.Spec.StorageClassName,
				"access_modes":   pvc.Spec.AccessModes,
				"capacity":       pvc.Status.Capacity.Storage().String(),
			},
		})
	}

	return resources, nil
}

// discoverStorageClasses discovers all StorageClasses
func (p *KubernetesProvider) discoverStorageClasses(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	storageClasses, err := p.client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, sc := range storageClasses.Items {
		tags := make(map[string]interface{})
		for k, v := range sc.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/sc/%s", p.clusterName, sc.Name),
			Name:     sc.Name,
			Type:     "kubernetes_storage_class",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: sc.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"provisioner":     sc.Provisioner,
				"reclaim_policy":  *sc.ReclaimPolicy,
				"volume_binding":  *sc.VolumeBindingMode,
				"allow_expansion": sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion,
			},
		})
	}

	return resources, nil
}

// discoverNetworkPolicies discovers all NetworkPolicies
func (p *KubernetesProvider) discoverNetworkPolicies(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	networkPolicies, err := p.client.NetworkingV1().NetworkPolicies(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, np := range networkPolicies.Items {
		tags := make(map[string]interface{})
		for k, v := range np.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, np.Namespace, np.Name),
			Name:     np.Name,
			Type:     "kubernetes_network_policy",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: np.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":      np.Namespace,
				"pod_selector":   np.Spec.PodSelector.MatchLabels,
				"policy_types":   np.Spec.PolicyTypes,
				"ingress_rules":  len(np.Spec.Ingress),
				"egress_rules":   len(np.Spec.Egress),
			},
		})
	}

	return resources, nil
}

// discoverRBACResources discovers RBAC resources (Roles, ClusterRoles, RoleBindings, ClusterRoleBindings)
func (p *KubernetesProvider) discoverRBACResources(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover Roles
	roles, err := p.client.RbacV1().Roles(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, role := range roles.Items {
		tags := make(map[string]interface{})
		for k, v := range role.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, role.Namespace, role.Name),
			Name:     role.Name,
			Type:     "kubernetes_role",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: role.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":   role.Namespace,
				"rules_count": len(role.Rules),
			},
		})
	}

	// Discover ClusterRoles
	clusterRoles, err := p.client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, cr := range clusterRoles.Items {
		tags := make(map[string]interface{})
		for k, v := range cr.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/cr/%s", p.clusterName, cr.Name),
			Name:     cr.Name,
			Type:     "kubernetes_cluster_role",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: cr.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"rules_count":       len(cr.Rules),
				"aggregation_rule":  cr.AggregationRule != nil,
			},
		})
	}

	// Discover RoleBindings
	roleBindings, err := p.client.RbacV1().RoleBindings(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, rb := range roleBindings.Items {
		tags := make(map[string]interface{})
		for k, v := range rb.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, rb.Namespace, rb.Name),
			Name:     rb.Name,
			Type:     "kubernetes_role_binding",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: rb.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":      rb.Namespace,
				"role_ref":       rb.RoleRef.Name,
				"subjects_count": len(rb.Subjects),
			},
		})
	}

	// Discover ClusterRoleBindings
	clusterRoleBindings, err := p.client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, crb := range clusterRoleBindings.Items {
		tags := make(map[string]interface{})
		for k, v := range crb.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/crb/%s", p.clusterName, crb.Name),
			Name:     crb.Name,
			Type:     "kubernetes_cluster_role_binding",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: crb.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"role_ref":       crb.RoleRef.Name,
				"subjects_count": len(crb.Subjects),
			},
		})
	}

	return resources, nil
}

// discoverHPAs discovers HorizontalPodAutoscalers
func (p *KubernetesProvider) discoverHPAs(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	hpas, err := p.client.AutoscalingV2().HorizontalPodAutoscalers(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, hpa := range hpas.Items {
		tags := make(map[string]interface{})
		for k, v := range hpa.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, hpa.Namespace, hpa.Name),
			Name:     hpa.Name,
			Type:     "kubernetes_hpa",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: hpa.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":        hpa.Namespace,
				"target_ref":       hpa.Spec.ScaleTargetRef.Name,
				"min_replicas":     *hpa.Spec.MinReplicas,
				"max_replicas":     hpa.Spec.MaxReplicas,
				"current_replicas": hpa.Status.CurrentReplicas,
				"desired_replicas": hpa.Status.DesiredReplicas,
			},
		})
	}

	return resources, nil
}

// discoverPDBs discovers PodDisruptionBudgets
func (p *KubernetesProvider) discoverPDBs(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pdbs, err := p.client.PolicyV1().PodDisruptionBudgets(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pdb := range pdbs.Items {
		tags := make(map[string]interface{})
		for k, v := range pdb.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, pdb.Namespace, pdb.Name),
			Name:     pdb.Name,
			Type:     "kubernetes_pdb",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: pdb.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":              pdb.Namespace,
				"min_available":          pdb.Spec.MinAvailable,
				"max_unavailable":        pdb.Spec.MaxUnavailable,
				"current_healthy":        pdb.Status.CurrentHealthy,
				"desired_healthy":        pdb.Status.DesiredHealthy,
				"disruptions_allowed":    pdb.Status.DisruptionsAllowed,
				"expected_pods":          pdb.Status.ExpectedPods,
			},
		})
	}

	return resources, nil
}

// GetRegions returns available Kubernetes clusters/contexts
func (p *KubernetesProvider) GetRegions() []string {
	// In Kubernetes context, regions are clusters/contexts
	return []string{p.clusterName}
}

// GetAccounts returns Kubernetes namespaces
func (p *KubernetesProvider) GetAccounts() []string {
	ctx := context.Background()
	namespaces, err := p.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return []string{}
	}

	var accounts []string
	for _, ns := range namespaces.Items {
		accounts = append(accounts, ns.Name)
	}
	return accounts
}

// DeleteResource deletes a Kubernetes resource
func (p *KubernetesProvider) DeleteResource(ctx context.Context, resourceID string) error {
	// Parse resource ID format: cluster/namespace/name or cluster/type/name
	parts := strings.Split(resourceID, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid resource ID format: %s", resourceID)
	}

	// Extract components
	namespace := ""
	resourceType := ""
	name := ""
	
	// Determine format based on parts
	if len(parts) == 3 {
		// Format: cluster/type/name (for cluster-scoped resources)
		resourceType = parts[1]
		name = parts[2]
	} else if len(parts) == 4 {
		// Format: cluster/namespace/type/name (for namespaced resources)
		namespace = parts[1]
		resourceType = parts[2]
		name = parts[3]
	} else {
		return fmt.Errorf("invalid resource ID format: %s", resourceID)
	}

	// Delete based on resource type
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	switch resourceType {
	case "deployment", "kubernetes_deployment":
		return p.client.AppsV1().Deployments(namespace).Delete(ctx, name, deleteOptions)
	case "service", "kubernetes_service":
		return p.client.CoreV1().Services(namespace).Delete(ctx, name, deleteOptions)
	case "pod", "kubernetes_pod":
		return p.client.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	case "configmap", "kubernetes_configmap":
		return p.client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, deleteOptions)
	case "secret", "kubernetes_secret":
		return p.client.CoreV1().Secrets(namespace).Delete(ctx, name, deleteOptions)
	case "ingress", "kubernetes_ingress":
		return p.client.NetworkingV1().Ingresses(namespace).Delete(ctx, name, deleteOptions)
	case "statefulset", "kubernetes_statefulset":
		return p.client.AppsV1().StatefulSets(namespace).Delete(ctx, name, deleteOptions)
	case "daemonset", "kubernetes_daemonset":
		return p.client.AppsV1().DaemonSets(namespace).Delete(ctx, name, deleteOptions)
	case "job", "kubernetes_job":
		return p.client.BatchV1().Jobs(namespace).Delete(ctx, name, deleteOptions)
	case "cronjob", "kubernetes_cronjob":
		return p.client.BatchV1().CronJobs(namespace).Delete(ctx, name, deleteOptions)
	case "pvc", "kubernetes_persistent_volume_claim":
		return p.client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, deleteOptions)
	case "pv", "kubernetes_persistent_volume":
		return p.client.CoreV1().PersistentVolumes().Delete(ctx, name, deleteOptions)
	case "namespace":
		return p.client.CoreV1().Namespaces().Delete(ctx, name, deleteOptions)
	case "networkpolicy", "kubernetes_network_policy":
		return p.client.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, deleteOptions)
	case "serviceaccount", "kubernetes_service_account":
		return p.client.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, deleteOptions)
	case "hpa", "kubernetes_hpa":
		return p.client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, deleteOptions)
	default:
		return fmt.Errorf("delete not supported for resource type: %s", resourceType)
	}
}

// Helper functions

func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getSecretKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getPodRestartCount(pod *corev1.Pod) int32 {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}

// ServiceAccounts and other helper methods
func (p *KubernetesProvider) discoverServiceAccounts(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	serviceAccounts, err := p.client.CoreV1().ServiceAccounts(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, sa := range serviceAccounts.Items {
		tags := make(map[string]interface{})
		for k, v := range sa.Labels {
			tags[k] = v
		}

		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("%s/%s/%s", p.clusterName, sa.Namespace, sa.Name),
			Name:     sa.Name,
			Type:     "kubernetes_service_account",
			Provider: "kubernetes",
			Region:   p.clusterName,
			State:    "active",
			Tags:     tags,
			CreatedAt: sa.CreationTimestamp.Time,
			Updated: time.Now(),
			Attributes: map[string]interface{}{
				"namespace":     sa.Namespace,
				"secrets_count": len(sa.Secrets),
			},
		})
	}

	return resources, nil
}