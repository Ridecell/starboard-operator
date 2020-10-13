package reports

import (
	"context"
	"fmt"
	//"reflect"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	starboardv1alpha1 "github.com/aquasecurity/starboard/pkg/apis/aquasecurity/v1alpha1"
	"github.com/aquasecurity/starboard/pkg/find/vulnerabilities"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aquasecurity/starboard/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StoreInterface interface {
	Write(ctx context.Context, workload kube.Object, reports vulnerabilities.WorkloadVulnerabilities, containerImages kube.ContainerImages) error
	Read(ctx context.Context, workload kube.Object, containerImage string) (*starboardv1alpha1.VulnerabilityScanResult, error)
	HasVulnerabilityReports(ctx context.Context, owner kube.Object, containerImages kube.ContainerImages) (bool, error)
}

type Store struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewStore(client client.Client, scheme *runtime.Scheme) *Store {
	return &Store{
		client: client,
		scheme: scheme,
	}
}

func (s *Store) Write(ctx context.Context, workload kube.Object, reports vulnerabilities.WorkloadVulnerabilities, containerImages kube.ContainerImages) error {
	owner, err := s.getRuntimeObjectFor(ctx, workload)
	if err != nil {
		return err
	}

	for containerName, report := range reports {
		reportName := fmt.Sprintf("%s-%s-%s", strings.ToLower(string(workload.Kind)),
			workload.Name, containerName)

		vulnerabilityReport := &starboardv1alpha1.VulnerabilityReport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      reportName,
				Namespace: workload.Namespace,
				Labels: labels.Set{
					kube.LabelResourceKind:      string(workload.Kind),
					kube.LabelResourceName:      workload.Name,
					kube.LabelResourceNamespace: workload.Namespace,
					kube.LabelContainerName:     containerName,
				},
				Annotations: map[string]string{
					"starboard.container.imagehash": strings.Split(containerImages[containerName], " ")[1],
				},
			},
			Report: report,
		}
		err = controllerutil.SetOwnerReference(owner, vulnerabilityReport, s.scheme)
		if err != nil {
			return err
		}

		err := s.client.Create(ctx, vulnerabilityReport)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Read(ctx context.Context, workload kube.Object, containerImageHash string) (*starboardv1alpha1.VulnerabilityScanResult, error) {
	vulnerabilityList := &starboardv1alpha1.VulnerabilityReportList{}

	err := s.client.List(ctx, vulnerabilityList, client.MatchingLabels{}, client.InNamespace(workload.Namespace))
	if err != nil {
		return nil, err
	}

	for _, item := range vulnerabilityList.Items {
		if containerHash, ok := item.Annotations["starboard.container.imagehash"]; ok && containerHash == strings.Split(containerImageHash, " ")[1] {
			return &item.Report, nil
		}
	}
	return nil, nil
}

func (s *Store) getRuntimeObjectFor(ctx context.Context, workload kube.Object) (metav1.Object, error) {
	var obj runtime.Object
	switch workload.Kind {
	case kube.KindPod:
		obj = &corev1.Pod{}
	case kube.KindReplicaSet:
		obj = &appsv1.ReplicaSet{}
	case kube.KindReplicationController:
		obj = &corev1.ReplicationController{}
	case kube.KindDeployment:
		obj = &appsv1.Deployment{}
	case kube.KindStatefulSet:
		obj = &appsv1.StatefulSet{}
	case kube.KindDaemonSet:
		obj = &appsv1.DaemonSet{}
	case kube.KindCronJob:
		obj = &v1beta1.CronJob{}
	case kube.KindJob:
		obj = &batchv1.Job{}
	default:
		return nil, fmt.Errorf("unknown workload kind: %s", workload.Kind)
	}
	err := s.client.Get(ctx, types.NamespacedName{Name: workload.Name, Namespace: workload.Namespace}, obj)
	if err != nil {
		return nil, err
	}
	return obj.(metav1.Object), nil
}

func (s *Store) HasVulnerabilityReports(ctx context.Context, owner kube.Object, containerImages kube.ContainerImages) (bool, error) {

	for _, containerImageHash := range containerImages {
		vulnerabilityReport, err := s.Read(ctx, owner, containerImageHash)
		if vulnerabilityReport == nil {
			return false, err
		}
	}
	return true, nil
}
