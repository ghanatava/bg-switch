package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all progressive deployments",
	Long:  `Display a table of all progressive deployments in the namespace with their status.`,
	RunE:  runList,
}

var (
	allNamespaces bool
)

func init() {
	listCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List progressive deployments across all namespaces")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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

	var list *unstructured.UnstructuredList
	if allNamespaces {
		list, err = dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to list progressive deployments: %w", err)
	}

	if len(list.Items) == 0 {
		if allNamespaces {
			fmt.Println("No progressive deployments found in any namespace")
		} else {
			fmt.Printf("No progressive deployments found in namespace '%s'\n", namespace)
		}
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if allNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tPHASE\tSTEP\tPERCENTAGE\tHEALTH\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tPHASE\tSTEP\tPERCENTAGE\tHEALTH\tAGE")
	}

	for _, item := range list.Items {
		status, _, _ := unstructured.NestedMap(item.Object, "status")
		spec, _, _ := unstructured.NestedMap(item.Object, "spec")

		name := item.GetName()
		ns := item.GetNamespace()
		phase := getStringField(status, "phase")
		currentStep := getInt64Field(status, "currentStep")
		canaryPercentage := getInt64Field(status, "canaryPercentage")
		healthStatus := getStringField(status, "healthStatus")

		canarySteps := getInt64Slice(spec, "canarySteps")
		totalSteps := len(canarySteps)

		age := item.GetCreationTimestamp().String()

		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d\t%d%%\t%s\t%s\n",
				ns, name, phase, currentStep, totalSteps, canaryPercentage, healthStatus, age)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d/%d\t%d%%\t%s\t%s\n",
				name, phase, currentStep, totalSteps, canaryPercentage, healthStatus, age)
		}
	}

	w.Flush()
	return nil
}
