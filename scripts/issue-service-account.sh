#!/bin/bash
# ServiceAccount 생성
kubectl create serviceaccount config-updater

# Role 생성
kubectl apply -f role.yaml

# RoleBinding 생성
kubectl create rolebinding config-updater \
  --role=deployment-reader \
  --serviceaccount=default:config-updater