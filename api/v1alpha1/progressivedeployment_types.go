/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MetricThreshold defines a metric query and its threshold
type MetricThreshold struct {
	// Query is the PromQL query to execute
	Query string `json:"query"`

	// Threshold is the maximum acceptable value
	// +kubebuilder:validation:Type=number
	Threshold float64 `json:"threshold"`
}

// MetricsConfig Custom type
type MetricsConfig struct {
	// PrometheusURL is the Prometheus endpoint (optional, defaults to in-cluster)
	PrometheusURL string          `json:"prometheusUrl,omitempty"`
	ErrorRate     MetricThreshold `json:"errorRate,omitempty"`
	Latency       MetricThreshold `json:"latency,omitempty"`
}

// ProgressiveDeploymentSpec defines the desired state of ProgressiveDeployment
type ProgressiveDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of ProgressiveDeployment. Edit progressivedeployment_types.go to remove/update
	// +optional
	TargetDeployment string          `json:"targetDeployment"`
	CanarySteps      []int           `json:"canarySteps"`
	StepDuration     metav1.Duration `json:"stepDuration"`
	Metrics          MetricsConfig   `json:"metrics"`
	AutoPromote      bool            `json:"autoPromote"`
}

// ProgressiveDeploymentStatus defines the observed state of ProgressiveDeployment.
type ProgressiveDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ProgressiveDeployment resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +optional
	// +kubebuilder:validation:Enum=Initializing;Analyzing;Promoting;RollingBack;Completed;RolledBack;Failed
	Phase string `json:"phase,omitempty"`
	// CurrentStep is the current canary step index (0-based)
	CurrentStep int `json:"currentStep,omitempty"`
	// CanaryPercentage is the current traffic percentage going to canary
	CanaryPercentage int `json:"canaryPercentage,omitempty"`
	// CanaryDeployment is the name of the canary Deployment
	CanaryDeployment string `json:"canaryDeployment,omitempty"`
	// HealthStatus indicates if the canary is healthy
	// +kubebuilder:validation:Enum=Healthy;Unhealthy;Unknown
	HealthStatus string `json:"healthStatus,omitempty"`
	// Metrics contains the last observed metric values
	// +kubebuilder:validation:Type=object
	Metrics map[string]float64 `json:"metrics,omitempty"`
	// Conditions represent the latest available observations
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
	LastAnalysisTime *metav1.Time       `json:"lastAnalysisTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=pd
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Step",type=integer,JSONPath=`.status.currentStep`
// +kubebuilder:printcolumn:name="Canary%",type=integer,JSONPath=`.status.canaryPercentage`
// +kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.healthStatus`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProgressiveDeployment is the Schema for the progressivedeployments API
type ProgressiveDeployment struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ProgressiveDeployment
	// +required
	Spec ProgressiveDeploymentSpec `json:"spec"`

	// status defines the observed state of ProgressiveDeployment
	// +optional
	Status ProgressiveDeploymentStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ProgressiveDeploymentList contains a list of ProgressiveDeployment
type ProgressiveDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProgressiveDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProgressiveDeployment{}, &ProgressiveDeploymentList{})
}
