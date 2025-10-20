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

var statusCmd = &cobra.Command{
	Use:   "status [deployment-name]",
	Short: "Get the status of a progressive deployment",
	Long:  `Display detailed status information about a progressive deployment including phase, step, health, and metrics.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Define the GVR for ProgressiveDeployment
	gvr := schema.GroupVersionResource{
		Group:    "apps.my.domain",
		Version:  "v1alpha1",
		Resource: "progressivedeployments",
	}

	// Get the ProgressiveDeployment
	ctx := context.Background()
	pd, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get progressive deployment: %w", err)
	}

	// Display status
	displayStatus(pd)

	return nil
}

func displayStatus(pd *unstructured.Unstructured) {
	status, _, _ := unstructured.NestedMap(pd.Object, "status")
	spec, _, _ := unstructured.NestedMap(pd.Object, "spec")

	// Extract fields
	phase := getStringField(status, "phase")
	currentStep := getInt64Field(status, "currentStep")
	canaryPercentage := getInt64Field(status, "canaryPercentage")
	healthStatus := getStringField(status, "healthStatus")
	canaryDeployment := getStringField(status, "canaryDeployment")

	// Get canary steps from spec
	canarySteps := getInt64Slice(spec, "canarySteps")
	totalSteps := len(canarySteps)

	// Get metrics if available
	metrics, _, _ := unstructured.NestedMap(status, "metrics")

	// Print formatted output
	fmt.Println("┌─────────────────────────────────────────────────┐")
	fmt.Printf("│  Progressive Deployment: %-22s │\n", pd.GetName())
	fmt.Println("├─────────────────────────────────────────────────┤")
	fmt.Printf("│  Phase:           %-29s │\n", phase)
	fmt.Printf("│  Step:            %d/%d (%-2d%%)                    │\n", currentStep, totalSteps, canaryPercentage)
	fmt.Printf("│  Health:          %-29s │\n", healthStatus)

	if canaryDeployment != "" {
		fmt.Printf("│  Canary:          %-29s │\n", canaryDeployment)
	}

	if len(metrics) > 0 {
		fmt.Println("│                                                 │")
		fmt.Println("│  Metrics:                                       │")
		for key, value := range metrics {
			fmt.Printf("│    %-15s %.6f                    │\n", key+":", value)
		}
	}

	fmt.Println("└─────────────────────────────────────────────────┘")

	// Add action hint
	switch phase {
	case "Analyzing":
		fmt.Println("\n⏳ Analyzing metrics... waiting for step duration")
	case "Promoting":
		fmt.Println("\n⬆️  Promoting to next step")
	case "RollingBack":
		fmt.Println("\n⬅️  Rolling back to stable version")
	case "Completed":
		fmt.Println("\n✅ Deployment completed successfully!")
	case "RolledBack":
		fmt.Println("\n🔄 Deployment was rolled back")
	case "Failed":
		fmt.Println("\n❌ Deployment failed")
	}
}
