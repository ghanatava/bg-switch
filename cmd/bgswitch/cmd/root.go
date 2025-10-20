package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
	namespace  string
)

var rootCmd = &cobra.Command{
	Use:   "bgswitch",
	Short: "BGswitch CLI - Control progressive deployments",
	Long: `BGswitch CLI provides commands to manage progressive deployments in Kubernetes.
	
Examples:
  bgswitch status my-app
  bgswitch promote my-app
  bgswitch rollback my-app`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $HOME/.kube/config)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
}

func getKubeConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			homeDir, _ := os.UserHomeDir()
			kubeconfig = homeDir + "/.kube/config"
		}
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config, nil
}

func getKubeClient() (*kubernetes.Clientset, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// Helper functions for extracting fields from unstructured data
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt64Field(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		if i, ok := val.(int64); ok {
			return i
		}
	}
	return 0
}

func getInt64Slice(m map[string]interface{}, key string) []int64 {
	if val, ok := m[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]int64, len(slice))
			for i, v := range slice {
				if num, ok := v.(int64); ok {
					result[i] = num
				} else if num, ok := v.(float64); ok {
					result[i] = int64(num)
				}
			}
			return result
		}
	}
	return nil
}
