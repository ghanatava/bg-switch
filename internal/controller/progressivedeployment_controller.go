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
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/ghanatava/bg-switch/api/v1alpha1"
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

func (r *ProgressiveDeploymentReconciler) handleInitializing(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling Initializing phase")

	// TODO: Create canary deployment
	// TODO: Verify target deployment exists

	// For now, just move to Analyzing phase
	pd.Status.Phase = "Analyzing"
	pd.Status.CurrentStep = 0
	pd.Status.CanaryPercentage = pd.Spec.CanarySteps[0]

	if err := r.updateStatus(ctx, pd); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Moved to Analyzing phase", "step", pd.Status.CurrentStep, "percentage", pd.Status.CanaryPercentage)

	// Requeue to handle Analyzing phase
	return ctrl.Result{}, nil
}

func (r *ProgressiveDeploymentReconciler) handleAnalyzing(ctx context.Context, pd *appsv1alpha1.ProgressiveDeployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling Analyzing phase")

	// TODO: Check if stepDuration has elapsed
	// TODO: Query Prometheus metrics
	// TODO: Determine if healthy or unhealthy

	// For now, just move to Promoting phase (simulate healthy)
	pd.Status.Phase = "Promoting"
	pd.Status.HealthStatus = "Healthy"

	if err := r.updateStatus(ctx, pd); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Metrics healthy, moving to Promoting phase")

	// Requeue to handle Promoting phase
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
