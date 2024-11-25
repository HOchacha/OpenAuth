package main

import (
	k8s "OpenAuth/pkg/k8sQuery"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// SendConfigFile sends config file to the target IP
func SendConfigFile(targetIP string, configFile string) error {
	// Read config file
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Send POST request to /config endpoint
	url := fmt.Sprintf("http://%s/config", targetIP)
	resp, err := http.Post(url, "application/yaml", bytes.NewBuffer(content))
	if err != nil {
		return fmt.Errorf("failed to send config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("server returned error (status: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func main() {
	namespace := flag.String("namespace", "default", "Kubernetes namespace")
	deploymentName := flag.String("deployment", "", "Deployment name")
	configFile := flag.String("config", "", "Config file to send")
	usePodIP := flag.Bool("use-pod-ip", false, "Use pod IPs instead of service IP")
	flag.Parse()

	if *deploymentName == "" || *configFile == "" {
		log.Fatal("deployment name and config file are required")
	}

	// Initialize Kubernetes client
	client, err := k8s.NewK8sClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Find deployment
	deployment, err := client.FindDeployment(*namespace, *deploymentName)
	if err != nil {
		log.Fatalf("Failed to find deployment: %v", err)
	}

	// Get target IPs
	if *usePodIP {
		podIPs, err := client.GetPodIPs(deployment)
		if err != nil {
			log.Fatalf("Failed to get pod IPs: %v", err)
		}

		// Send config to each pod
		for _, ip := range podIPs {
			fmt.Printf("Sending config to pod IP: %s\n", ip)
			if err := SendConfigFile(ip, *configFile); err != nil {
				log.Printf("Failed to send config to %s: %v", ip, err)
			}
		}
	} else {
		// Get service IP
		serviceIP, err := client.GetServiceIP(deployment)
		if err != nil {
			log.Fatalf("Failed to get service IP: %v", err)
		}

		fmt.Printf("Sending config to service IP: %s\n", serviceIP)
		if err := SendConfigFile(serviceIP, *configFile); err != nil {
			log.Fatalf("Failed to send config: %v", err)
		}
	}
}
