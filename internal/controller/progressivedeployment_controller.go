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

package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/ghanatava/bg-switch/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

// ProgressiveDeploymentReconciler reconciles a ProgressiveDeployment object
type ProgressiveDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// updateStatus updates the ProgressiveDeployment status
func (r *ProgressiveDeploymentReconciler) updateStatus(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) error {
	return r.Status().Update(ctx, pd)
}

func (r *ProgressiveDeploymentReconciler) getTargetDeployment(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (*appsv1.Deployment, error) {
	log := logf.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: pd.Namespace,
		Name:      pd.Spec.TargetDeployment,
	}, deployment)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "Target deployment not found", "deployment", pd.Spec.TargetDeployment)
			return nil, err
		}
		log.Error(err, "Failed to get target deployment")
		return nil, err
	}

	log.Info("Found target deployment", "name", deployment.Name, "replicas", *deployment.Spec.Replicas)
	return deployment, nil
}

// createCanaryDeployment creates a canary Deployment as a clone of the target
func (r *ProgressiveDeploymentReconciler) createCanaryDeployment(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment, targetDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	log := logf.FromContext(ctx)

	// Generate canary deployment name
	canaryName := fmt.Sprintf("%s-canary", pd.Spec.TargetDeployment)

	// Clone the target deployment spec
	canary := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canaryName,
			Namespace: pd.Namespace,
			Labels: map[string]string{
				"app":                    targetDeployment.Labels["app"],
				"progressive-deployment": pd.Name,
				"deployment-type":        "canary",
			},
		},
		Spec: *targetDeployment.Spec.DeepCopy(),
	}

	// Update canary pod labels to differentiate from stable
	if canary.Spec.Template.Labels == nil {
		canary.Spec.Template.Labels = make(map[string]string)
	}
	canary.Spec.Template.Labels["version"] = "canary"
	canary.Spec.Template.Labels["deployment-type"] = "canary"

	// Update selector to match new labels
	if canary.Spec.Selector == nil {
		canary.Spec.Selector = &metav1.LabelSelector{}
	}
	if canary.Spec.Selector.MatchLabels == nil {
		canary.Spec.Selector.MatchLabels = make(map[string]string)
	}
	canary.Spec.Selector.MatchLabels["app"] = targetDeployment.Labels["app"]
	canary.Spec.Selector.MatchLabels["version"] = "canary"

	// Start with 0 replicas - we'll adjust based on canary percentage
	replicas := int32(0)
	canary.Spec.Replicas = &replicas

	// Set owner reference so canary gets deleted when ProgressiveDeployment is deleted
	if err := ctrl.SetControllerReference(pd, canary, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference")
		return nil, err
	}

	// Create the canary deployment
	if err := r.Create(ctx, canary); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Canary deployment already exists", "name", canaryName)
			// Fetch existing canary
			existingCanary := &appsv1.Deployment{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: pd.Namespace, Name: canaryName}, existingCanary); err != nil {
				return nil, err
			}
			return existingCanary, nil
		}
		log.Error(err, "Failed to create canary deployment")
		return nil, err
	}

	log.Info("Created canary deployment", "name", canaryName, "replicas", 0)
	return canary, nil
}

// handleInitializing creates the canary deployment
func (r *ProgressiveDeploymentReconciler) handleInitializing(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling Initializing phase")

	// Step 1: Get the target deployment
	targetDeployment, err := r.getTargetDeployment(ctx, pd)
	if err != nil {
		// Update status to Failed
		pd.Status.Phase = "Failed"
		pd.Status.HealthStatus = "Unknown"
		if updateErr := r.updateStatus(ctx, pd); updateErr != nil {
			log.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Step 2: Create canary deployment
	canary, err := r.createCanaryDeployment(ctx, pd, targetDeployment)
	if err != nil {
		// Update status to Failed
		pd.Status.Phase = "Failed"
		pd.Status.HealthStatus = "Unknown"
		if updateErr := r.updateStatus(ctx, pd); updateErr != nil {
			log.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Step 3: Update status
	pd.Status.Phase = "Analyzing"
	pd.Status.CurrentStep = 0
	pd.Status.CanaryPercentage = pd.Spec.CanarySteps[0]
	pd.Status.CanaryDeployment = canary.Name
	pd.Status.HealthStatus = "Unknown"

	if err := r.updateStatus(ctx, pd); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Moved to Analyzing phase",
		"step", pd.Status.CurrentStep,
		"percentage", pd.Status.CanaryPercentage,
		"canary", pd.Status.CanaryDeployment)

	return ctrl.Result{}, nil
}

// handleAnalyzing waits for stepDuration and checks metrics
func (r *ProgressiveDeploymentReconciler) handleAnalyzing(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling Analyzing phase")

	stepDuration := pd.Spec.StepDuration.Duration
	now := metav1.Now()

	// If LastAnalysisTime is not set, this is the first time - adjust traffic and wait
	if pd.Status.LastAnalysisTime == nil {
		log.Info("Starting analysis period", "duration", stepDuration, "canaryPercentage", pd.Status.CanaryPercentage)

		// Get target deployment
		targetDeployment, err := r.getTargetDeployment(ctx, pd)
		if err != nil {
			log.Error(err, "Failed to get target deployment for traffic shifting")
			return ctrl.Result{}, err
		}

		// Adjust traffic based on current canary percentage
		if err := r.adjustTraffic(ctx, pd, targetDeployment); err != nil {
			log.Error(err, "Failed to adjust traffic")
			return ctrl.Result{}, err
		}

		// Set analysis start time
		pd.Status.LastAnalysisTime = &now
		if err := r.updateStatus(ctx, pd); err != nil {
			return ctrl.Result{}, err
		}

		// Wait for stepDuration before analyzing metrics
		log.Info("Traffic adjusted, waiting for stabilization", "duration", stepDuration)
		return ctrl.Result{RequeueAfter: stepDuration}, nil
	}

	// Check if enough time has elapsed
	elapsed := now.Sub(pd.Status.LastAnalysisTime.Time)
	if elapsed < stepDuration {
		remaining := stepDuration - elapsed
		log.Info("Still analyzing", "elapsed", elapsed, "remaining", remaining)
		return ctrl.Result{RequeueAfter: remaining}, nil
	}

	// Enough time passed - analyze metrics and decide
	log.Info("Analysis period complete", "elapsed", elapsed)

	// TODO: Query Prometheus metrics here
	// For now, assume healthy

	// Metrics healthy - move to Promoting
	pd.Status.Phase = "Promoting"
	pd.Status.HealthStatus = "Healthy"
	pd.Status.LastAnalysisTime = nil // Reset for next step

	if err := r.updateStatus(ctx, pd); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Metrics healthy, moving to Promoting phase")
	return ctrl.Result{}, nil
}

// handlePromoting moves to the next canary step
func (r *ProgressiveDeploymentReconciler) handlePromoting(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling Promoting phase")

	// Check if we're at the last step
	if pd.Status.CurrentStep >= len(pd.Spec.CanarySteps)-1 {
		// Deployment complete!
		log.Info("All steps completed successfully")
		pd.Status.Phase = "Completed"
		pd.Status.CanaryPercentage = 100

		if err := r.updateStatus(ctx, pd); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Move to next step
	pd.Status.CurrentStep++
	pd.Status.CanaryPercentage = pd.Spec.CanarySteps[pd.Status.CurrentStep]
	pd.Status.Phase = "Analyzing"

	if err := r.updateStatus(ctx, pd); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Promoted to next step", "step", pd.Status.CurrentStep, "percentage", pd.Status.CanaryPercentage)

	// Requeue to analyze the new step
	return ctrl.Result{Requeue: true}, nil
}

func (r *ProgressiveDeploymentReconciler) handleRollingBack(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling RollingBack phase")

	// TODO: Restore stable deployment
	// TODO: Delete canary deployment

	// For now, just mark as RolledBack
	pd.Status.Phase = "RolledBack"
	pd.Status.CanaryPercentage = 0

	if err := r.updateStatus(ctx, pd); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Rollback completed")

	return ctrl.Result{}, nil
}

// calculateReplicaDistribution calculates stable and canary replica counts
func calculateReplicaDistribution(totalReplicas int, canaryPercentage int) (stableReplicas, canaryReplicas int32) {
	if canaryPercentage <= 0 {
		return int32(totalReplicas), 0
	}

	if canaryPercentage >= 100 {
		return 0, int32(totalReplicas)
	}

	// Calculate canary replicas (round up to ensure traffic gets through)
	canaryFloat := float64(totalReplicas) * float64(canaryPercentage) / 100.0
	canaryReplicas = int32(math.Ceil(canaryFloat))

	// Remaining go to stable
	stableReplicas = int32(totalReplicas) - canaryReplicas

	// Ensure we don't go negative
	if stableReplicas < 0 {
		stableReplicas = 0
	}
	if canaryReplicas < 0 {
		canaryReplicas = 0
	}

	return stableReplicas, canaryReplicas
}

// adjustTraffic adjusts replica counts for stable and canary deployments
func (r *ProgressiveDeploymentReconciler) adjustTraffic(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment, targetDeployment *appsv1.Deployment) error {
	log := logf.FromContext(ctx)

	// Get total desired replicas from target
	totalReplicas := int(*targetDeployment.Spec.Replicas)

	// Calculate distribution
	stableReplicas, canaryReplicas := calculateReplicaDistribution(totalReplicas, pd.Status.CanaryPercentage)

	log.Info("Calculating traffic distribution",
		"total", totalReplicas,
		"canaryPercentage", pd.Status.CanaryPercentage,
		"stable", stableReplicas,
		"canary", canaryReplicas)

	// Update stable deployment (target)
	targetDeployment.Spec.Replicas = &stableReplicas
	if err := r.Update(ctx, targetDeployment); err != nil {
		log.Error(err, "Failed to update stable deployment replicas")
		return err
	}
	log.Info("Updated stable deployment", "replicas", stableReplicas)

	// Update canary deployment
	canaryDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: pd.Namespace,
		Name:      pd.Status.CanaryDeployment,
	}, canaryDeployment); err != nil {
		log.Error(err, "Failed to get canary deployment")
		return err
	}

	canaryDeployment.Spec.Replicas = &canaryReplicas
	if err := r.Update(ctx, canaryDeployment); err != nil {
		log.Error(err, "Failed to update canary deployment replicas")
		return err
	}
	log.Info("Updated canary deployment", "replicas", canaryReplicas)

	return nil
}

// +kubebuilder:rbac:groups=apps.my.domain,resources=progressivedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.my.domain,resources=progressivedeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.my.domain,resources=progressivedeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProgressiveDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *ProgressiveDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	// Step 1: Fetch the ProgressiveDeployment instance
	var progressiveDeployment appsv1alpha1.ProgressiveDeployment
	if err := r.Get(ctx, req.NamespacedName, &progressiveDeployment); err != nil {
		if errors.IsNotFound(err) {
			// Resource deleted, nothing to do
			log.Info("ProgressiveDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue
		log.Error(err, "Failed to get ProgressiveDeployment")
		return ctrl.Result{}, err
	}
	log.Info("Reconciling ProgressiveDeployment",
		"name", progressiveDeployment.Name,
		"phase", progressiveDeployment.Status.Phase,
		"step", progressiveDeployment.Status.CurrentStep)

	// Step 2: Initialize status if this is a new resource
	if progressiveDeployment.Status.Phase == "" {
		log.Info("Initializing new ProgressiveDeployment")
		progressiveDeployment.Status.Phase = "Initializing"
		progressiveDeployment.Status.CurrentStep = 0
		progressiveDeployment.Status.CanaryPercentage = 0
		progressiveDeployment.Status.HealthStatus = "Unknown"

		if err := r.updateStatus(ctx, &progressiveDeployment); err != nil {
			log.Error(err, "Failed to initialize status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// Step 3: State machine - handle current phase
	switch progressiveDeployment.Status.Phase {

	case "Initializing":
		return r.handleInitializing(ctx, &progressiveDeployment)

	case "Analyzing":
		return r.handleAnalyzing(ctx, &progressiveDeployment)

	case "Promoting":
		return r.handlePromoting(ctx, &progressiveDeployment)

	case "RollingBack":
		return r.handleRollingBack(ctx, &progressiveDeployment)

	case "Completed", "RolledBack", "Failed":
		// Terminal states - nothing to do
		log.Info("ProgressiveDeployment in terminal state", "phase", progressiveDeployment.Status.Phase)
		return ctrl.Result{}, nil

	default:
		// Unknown phase - reset to Initializing
		log.Info("Unknown phase, resetting to Initializing", "phase", progressiveDeployment.Status.Phase)
		progressiveDeployment.Status.Phase = "Initializing"
		if err := r.updateStatus(ctx, &progressiveDeployment); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProgressiveDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.ProgressiveDeployment{}).
		Named("progressivedeployment").
		Complete(r)
}
