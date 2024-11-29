package k8sQuery

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"time"
)

type K8sClient struct {
	clientset *kubernetes.Clientset
}

/*
Package k8sQuery implements a Kubernetes client wrapper for managing cluster resources.

Implemented Functions:

# Client Initialization
- NewK8sClient(): Creates new Kubernetes client instance

# Deployment Operations
- FindDeployment(namespace, name): Finds specific deployment
- CreateDeployment(namespace, deployment): Creates new deployment
- UpdateDeployment(namespace, deployment): Updates existing deployment
- DeleteDeployment(namespace, name): Deletes deployment
- ListDeployments(namespace): Lists all deployments in namespace
- CreateBasicDeployment(namespace, name, image, port, replicas): Creates deployment with basic configuration
- IsDeploymentExist(namespace, name): Checks if deployment exists
- WaitForDeploymentReady(namespace, name, timeout): Waits until deployment is ready

# Pod Operations
- CreatePod(namespace, pod): Creates new pod
- GetPod(namespace, name): Gets specific pod
- DeletePod(namespace, name): Deletes pod
- ListPods(namespace): Lists all pods in namespace
- GetPodIPs(deployment): Gets IPs of all pods in deployment

# Service Operations
- CreateService(namespace, service): Creates new service
- GetService(namespace, name): Gets specific service
- UpdateService(namespace, service): Updates existing service
- DeleteService(namespace, name): Deletes service
- ListServices(namespace): Lists all services in namespace
- CreateBasicService(namespace, name, port, targetPort): Creates service with basic configuration
- GetServiceIP(deployment): Gets IP of service associated with deployment
- GetServiceEndpoint(namespace, name): Gets service endpoint
- IsServiceExist(namespace, name): Checks if service exists

# Query Operations
- GetPodIPs: Gets IPs of all pods in deployment
- GetServiceIP: Gets service IP associated with deployment

This package provides a simplified interface for managing Kubernetes resources,
handling common operations for Deployments, Pods, and Services.
*/

// NewK8sClient creates a new Kubernetes client
func NewK8sClient() (*K8sClient, error) {
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return &K8sClient{clientset: clientset}, nil
}

// FindDeployment finds a specific deployment
func (c *K8sClient) FindDeployment(namespace, name string) (*appsv1.Deployment, error) {
	deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(
		context.TODO(),
		name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %v", err)
	}
	return deployment, nil
}

// GetPodIPs gets IPs of all pods in the deployment
func (c *K8sClient) GetPodIPs(deployment *appsv1.Deployment) ([]string, error) {
	// Get label selector from deployment
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)

	// List pods with the deployment's labels
	pods, err := c.clientset.CoreV1().Pods(deployment.Namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	var ips []string
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			ips = append(ips, pod.Status.PodIP)
		}
	}
	return ips, nil
}

// GetServiceIP gets the IP of the service associated with the deployment
func (c *K8sClient) GetServiceIP(deployment *appsv1.Deployment) (string, error) {
	// Get services in the same namespace
	services, err := c.clientset.CoreV1().Services(deployment.Namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to list services: %v", err)
	}

	// Find service that matches deployment labels
	for _, service := range services.Items {
		// Check if service selector matches deployment labels
		matches := true
		for selectorKey, selectorValue := range service.Spec.Selector {
			if labelValue, exists := deployment.Spec.Template.Labels[selectorKey]; !exists || labelValue != selectorValue {
				matches = false
				break
			}
		}
		if matches {
			// Return ClusterIP for the matching service
			return service.Spec.ClusterIP, nil
		}
	}

	return "", fmt.Errorf("no matching service found for deployment")
}

// Deployment 관련 함수들
func (c *K8sClient) CreateDeployment(namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Create(
		context.TODO(),
		deployment,
		metav1.CreateOptions{},
	)
}

func (c *K8sClient) UpdateDeployment(namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Update(
		context.TODO(),
		deployment,
		metav1.UpdateOptions{},
	)
}

func (c *K8sClient) DeleteDeployment(namespace, name string) error {
	return c.clientset.AppsV1().Deployments(namespace).Delete(
		context.TODO(),
		name,
		metav1.DeleteOptions{},
	)
}

func (c *K8sClient) ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	return c.clientset.AppsV1().Deployments(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
}

// Pod 관련 함수들
func (c *K8sClient) CreatePod(namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Create(
		context.TODO(),
		pod,
		metav1.CreateOptions{},
	)
}

func (c *K8sClient) GetPod(namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(
		context.TODO(),
		name,
		metav1.GetOptions{},
	)
}

func (c *K8sClient) DeletePod(namespace, name string) error {
	return c.clientset.CoreV1().Pods(namespace).Delete(
		context.TODO(),
		name,
		metav1.DeleteOptions{},
	)
}

func (c *K8sClient) ListPods(namespace string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
}

// Service 관련 함수들
func (c *K8sClient) CreateService(namespace string, service *corev1.Service) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Create(
		context.TODO(),
		service,
		metav1.CreateOptions{},
	)
}

func (c *K8sClient) GetService(namespace, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(
		context.TODO(),
		name,
		metav1.GetOptions{},
	)
}

func (c *K8sClient) UpdateService(namespace string, service *corev1.Service) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Update(
		context.TODO(),
		service,
		metav1.UpdateOptions{},
	)
}

func (c *K8sClient) DeleteService(namespace, name string) error {
	return c.clientset.CoreV1().Services(namespace).Delete(
		context.TODO(),
		name,
		metav1.DeleteOptions{},
	)
}

func (c *K8sClient) ListServices(namespace string) (*corev1.ServiceList, error) {
	return c.clientset.CoreV1().Services(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
}

// 유틸리티 함수들
func (c *K8sClient) WaitForDeploymentReady(namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s to be ready", name)
		default:
			deployment, err := c.FindDeployment(namespace, name)
			if err != nil {
				return err
			}

			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				return nil
			}

			time.Sleep(2 * time.Second)
		}
	}
}

// CreateBasicDeployment creates a deployment with common defaults
func (c *K8sClient) CreateBasicDeployment(namespace, name, image string, port int32, replicas int32) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
								},
							},
						},
					},
				},
			},
		},
	}

	return c.CreateDeployment(namespace, deployment)
}

// CreateBasicService creates a service with common defaults
func (c *K8sClient) CreateBasicService(namespace, name string, port, targetPort int32) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
		},
	}

	return c.CreateService(namespace, service)
}

// GetServiceEndpoint returns the cluster-internal endpoint for a service
func (c *K8sClient) GetServiceEndpoint(namespace, name string) (string, error) {
	service, err := c.GetService(namespace, name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace), nil
}

// IsDeploymentExist checks if a deployment exists
func (c *K8sClient) IsDeploymentExist(namespace, name string) (bool, error) {
	_, err := c.FindDeployment(namespace, name)
	if err != nil {
		if statusError, ok := err.(*errors.StatusError); ok && statusError.Status().Code == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsServiceExist checks if a service exists
func (c *K8sClient) IsServiceExist(namespace, name string) (bool, error) {
	_, err := c.GetService(namespace, name)
	if err != nil {
		if statusError, ok := err.(*errors.StatusError); ok && statusError.Status().Code == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
