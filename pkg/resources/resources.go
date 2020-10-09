package resources

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aquasecurity/starboard/pkg/kube"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func GetContainerImagesFromPodStatus(status corev1.PodStatus) kube.ContainerImages {
	images := kube.ContainerImages{}
	for _, container := range status.ContainerStatuses {
		// Extract Image hash from Image ID
		imageIdSlice := strings.Split(container.ImageID, ":")
		images[container.Name] = imageIdSlice[len(imageIdSlice)-1]
	}
	return images
}

func GetContainerImagesFromJob(job *batchv1.Job) (kube.ContainerImages, error) {
	var containerImagesAsJSON string
	var ok bool

	if containerImagesAsJSON, ok = job.Annotations[kube.AnnotationContainerImages]; !ok {
		return nil, fmt.Errorf("job does not have required annotation: %s", kube.AnnotationContainerImages)
	}
	containerImages := kube.ContainerImages{}
	err := containerImages.FromJSON(containerImagesAsJSON)
	if err != nil {
		return nil, fmt.Errorf("parsing job annotation: %s: %w", kube.AnnotationContainerImages, err)
	}
	return containerImages, nil
}

// HasContainersReadyCondition iterates conditions of the specified Pod to check
// whether all containers in the Pod are ready.
func HasContainersReadyCondition(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.ContainersReady {
			return true
		}
	}
	return false
}

// GetImmediateOwnerReference returns the immediate owner of the specified Pod.
// For example, for a Pod controlled by a Deployment it will return the active ReplicaSet object,
// whereas for an unmanaged Pod the immediate owner is the Pod itself.
func GetImmediateOwnerReference(pod *corev1.Pod) kube.Object {
	ownerRef := metav1.GetControllerOf(pod)
	if ownerRef != nil {
		return kube.Object{
			Namespace: pod.Namespace,
			Kind:      kube.Kind(ownerRef.Kind),
			Name:      ownerRef.Name,
		}
	}
	return kube.Object{
		Namespace: pod.Namespace,
		Kind:      kube.KindPod,
		Name:      pod.Name,
	}
}
