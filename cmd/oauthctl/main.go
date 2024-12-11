package main

import (
	k8s "OpenAuth/pkg/k8sQuery"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"

	"log"
	"net/http"
	"time"
)

// SendConfigFile sends config file to the target IP
func SendConfigFile(targetIP string, configFile string, token string) error {
	log.Printf("Starting to send config file '%s' to IP: %s", configFile, targetIP)

	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
		return fmt.Errorf("failed to read config file: %v", err)
	}
	log.Printf("Successfully read config file, size: %d bytes", len(content))

	url := fmt.Sprintf("http://%s:8080/config", targetIP)
	log.Printf("Attempting POST request to URL: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			log.Printf("Response Status: %s", resp.Status)
			log.Printf("Response Headers: %v", resp.Header)
			body, readErr := ioutil.ReadAll(resp.Body)
			if readErr != nil {
				log.Printf("Error reading response body: %v", readErr)
			} else {
				log.Printf("Response Body: %s", string(body))
			}
			resp.Body.Close()
		}
		log.Printf("HTTP POST request failed after %v: %v", time.Since(startTime), err)
		return fmt.Errorf("failed to send config: %v", err)
	}
	log.Printf("Request completed in %v", time.Since(startTime))
	defer resp.Body.Close()

	log.Printf("Response Status: %s", resp.Status)
	log.Printf("Response Headers: %v", resp.Header)

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return fmt.Errorf("failed to read error response: %v", err)
		}
		log.Printf("Server returned non-OK status. Body: %s", string(body))
		return fmt.Errorf("server returned error (status: %d): %s", resp.StatusCode, string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading success response body: %v", err)
	} else {
		log.Printf("Server response body: %s", string(body))
	}

	log.Printf("Successfully sent config to %s", targetIP)
	return nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	namespace := flag.String("namespace", "default", "Kubernetes namespace")
	configFile := flag.String("config", "", "Config file to send")
	usePodIP := flag.Bool("use-pod-ip", false, "Use pod IPs instead of service IP")
	saName := flag.String("serviceaccount", "oauth-configurator", "ServiceAccount name")

	log.Printf("Starting config sender application")
	flag.Parse()

	log.Printf("Parsed flags: namespace=%s, configFile=%s, usePodIP=%v",
		*namespace, *configFile, *usePodIP)

	if *configFile == "" {
		log.Fatal("config file is required")
	}

	// Initialize Kubernetes client
	log.Printf("Initializing Kubernetes client")
	client, err := k8s.NewK8sClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	log.Printf("Successfully initialized Kubernetes client")

	log.Printf("Namespace: %s, saName: %s", *namespace, *saName)
	// ServiceAccount 토큰 가져오기
	token, err := client.GetServiceAccountToken(*namespace, *saName)
	if err != nil {
		log.Fatalf("Failed to get ServiceAccount token: %v", err)
	}
	log.Printf("Successfully retrieved ServiceAccount token")

	// Find deployment
	log.Printf("Looking for deployment 'openauth' in namespace '%s'", *namespace)
	deployment, err := client.FindDeployment(*namespace, "openauth")
	if err != nil {
		log.Fatalf("Failed to find deployment: %v", err)
	}
	log.Printf("Found deployment: %s", deployment.Name)

	// Get target IPs
	if *usePodIP {
		log.Printf("Retrieving Pod IPs for deployment")
		podIPs, err := client.GetPodIPs(deployment)
		if err != nil {
			log.Fatalf("Failed to get pod IPs: %v", err)
		}
		log.Printf("Found %d pod IPs: %v", len(podIPs), podIPs)

		// Send config to each pod
		for i, ip := range podIPs {
			log.Printf("Processing pod %d/%d with IP: %s", i+1, len(podIPs), ip)
			if err := SendConfigFile(ip, *configFile, token); err != nil {
				log.Printf("Failed to send config to %s: %v", ip, err)
			}
		}
	} else {
		log.Printf("Retrieving Service IP for deployment")

		serviceIP, err := client.GetServiceIP(deployment)
		if err != nil {
			log.Fatalf("Failed to get service IP: %v", err)
		}
		log.Printf("Found service IP: %s", serviceIP)

		if err := SendConfigFile(serviceIP, *configFile, token); err != nil {
			log.Fatalf("Failed to send config: %v", err)
		}
	}

	log.Printf("Config sender application completed successfully")
}
