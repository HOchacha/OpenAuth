package main

import (
	"OpenAuth/pkg/configServer"

	"log"
)

func main() {
	// 설정 예시
	config := configServer.Config{
		Routes: []configServer.RouteConfig{
			{
				Path:   "/api/test",
				Method: "POST",
				// RequestFilter 체인
				RequestFilters: []configServer.RequestFilter{
					{
						RemoteServer: "http://10.244.137.157:5000/otp/signup",
						RequestFormat: map[string]string{
							"Content-Type": "application/json",
						},
						FieldsToSend: []string{"username"},
					},
					{
						RemoteServer: "http://10.244.137.157:5000/otp/signup",
						RequestFormat: map[string]string{
							"Content-Type": "application/json",
						},
						FieldsToSend: []string{"username"},
					},
					{
						RemoteServer: "http://10.244.137.157:5000/otp/signup",
						RequestFormat: map[string]string{
							"Content-Type": "application/json",
						},
						FieldsToSend: []string{"username"},
					},
				},
			},
		},
	}

	router := configServer.SetupRouter(config)

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
