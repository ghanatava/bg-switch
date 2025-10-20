package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var promoteCmd = &cobra.Command{
	Use:   "promote [deployment-name]",
	Short: "Manually promote to the next canary step",
	Long:  `Promote the progressive deployment to the next canary step. Only works when autoPromote is false.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPromote,
}

func init() {
	rootCmd.AddCommand(promoteCmd)
}

func runPromote(cmd *cobra.Command, args []string) error {
	deploymentName := args[0]

	// Get dynamic client
	config, err := getKubeConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define the GVR
	gvr := schema.GroupVersionResource{
		Group:    "apps.my.domain",
		Version:  "v1alpha1",
		Resource: "progressivedeployments",
	}

	ctx := context.Background()

	// Get the ProgressiveDeployment
	pd, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get progressive deployment: %w", err)
	}

	// Check current phase
	status, _, _ := unstructured.NestedMap(pd.Object, "status")
	phase := getStringField(status, "phase")
	currentStep := getInt64Field(status, "currentStep")

	// Get spec
	spec, _, _ := unstructured.NestedMap(pd.Object, "spec")
	canarySteps := getInt64Slice(spec, "canarySteps")
	autoPromote, _, _ := unstructured.NestedBool(spec, "autoPromote")

	// Validation
	if autoPromote {
		fmt.Println("⚠️  Warning: autoPromote is enabled. The operator will automatically promote.")
		fmt.Println("   Consider setting autoPromote: false for manual control.")
	}

	if phase != "Analyzing" && phase != "Promoting" {
		return fmt.Errorf("cannot promote in phase '%s'. Must be in 'Analyzing' or 'Promoting' phase", phase)
	}

	if int(currentStep) >= len(canarySteps)-1 {
		return fmt.Errorf("already at final step (%d/%d)", currentStep+1, len(canarySteps))
	}

	// Manual promotion: Move to Promoting phase
	// The operator will then advance the step
	if phase == "Analyzing" {
		unstructured.SetNestedField(pd.Object, "Promoting", "status", "phase")

		_, err = dynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(ctx, pd, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}

		fmt.Printf("✅ Promoted %s to next step\n", deploymentName)
		fmt.Printf("   Moving from step %d to step %d\n", currentStep, currentStep+1)
	} else {
		fmt.Println("ℹ️  Already promoting...")
	}

	return nil
}
