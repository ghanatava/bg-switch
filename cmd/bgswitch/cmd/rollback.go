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

var rollbackCmd = &cobra.Command{
	Use:   "rollback [deployment-name]",
	Short: "Manually rollback to the stable version",
	Long:  `Immediately rollback the progressive deployment to the stable version, scaling down the canary.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRollback,
}

var (
	force bool
)

func init() {
	rollbackCmd.Flags().BoolVar(&force, "force", false, "Force rollback without confirmation")
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
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

	// Validation
	if phase == "RolledBack" {
		fmt.Println("ℹ️  Deployment is already rolled back")
		return nil
	}

	if phase == "Completed" {
		return fmt.Errorf("cannot rollback a completed deployment")
	}

	if phase == "Failed" {
		fmt.Println("⚠️  Deployment is in Failed state")
	}

	// Confirmation
	if !force {
		fmt.Printf("⚠️  About to rollback '%s' (currently in %s phase)\n", deploymentName, phase)
		fmt.Print("Continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Rollback cancelled")
			return nil
		}
	}

	// Trigger rollback by setting phase to RollingBack
	unstructured.SetNestedField(pd.Object, "RollingBack", "status", "phase")
	unstructured.SetNestedField(pd.Object, "Unhealthy", "status", "healthStatus")

	_, err = dynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(ctx, pd, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	fmt.Printf("🔄 Rollback initiated for %s\n", deploymentName)
	fmt.Println("   The operator will restore the stable deployment")

	return nil
}
