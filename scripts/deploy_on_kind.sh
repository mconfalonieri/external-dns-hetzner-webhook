#!/usr/bin/bash

set -e # exit on first error

# local registry
LOCAL_REGISTRY_PORT=5001
LOCAL_REGISTRY_NAME=kind-registry
LOCAL_REGISTRY_RUNNING=$(docker ps -a | grep -q $LOCAL_REGISTRY_NAME && echo "true" || echo "false")

# docker
IMAGE_EXTERNAL_DNS_WEBHOOK_PROVIDER=ghcr.io/ionos-cloud/external-dns-webhook-provider:latest

IMAGE_REGISTRY=localhost:$LOCAL_REGISTRY_PORT
IMAGE_NAME=external-dns-ionos-webhook
IMAGE=$IMAGE_REGISTRY/$IMAGE_NAME

#kind
KIND_CLUSTER_NAME=external-dns
KIND_CLUSTER_CONFIG=./deployments/kind/cluster.yaml
KIND_CLUSTER_RUNNING=$(kind get clusters -q | grep -q $KIND_CLUSTER_NAME && echo "true" || echo "false")
KIND_CLUSTER_WAIT=60s

## helm
HELM_CHART=bitnami/external-dns
HELM_RELEASE_NAME=ionos-external-dns
HELM_VALUES_FILE=deployments/helm/local-kind-values.yaml

HELM_CHART_REPO_URL=https://charts.bitnami.com/bitnami
HELM_CHART_REPO_NAME=bitnami

# if there is a clean up argument, delete the kind cluster and local registry
if [ "$1" = "clean" ]; then
    printf "Cleaning up...\n"
    if [ "$KIND_CLUSTER_RUNNING" = "true" ]; then
        printf "Deleting kind cluster...\n"
        kind delete cluster --name "$KIND_CLUSTER_NAME"
    fi
    if [ "$LOCAL_REGISTRY_RUNNING" = "true" ]; then
        printf "Deleting local registry...\n"
        docker rm -f "$LOCAL_REGISTRY_NAME"
    fi
    exit 0
fi

# if there is a helm-delete argument, delete the helm release
if [ "$1" = "helm-delete" ]; then
    printf "Deleting helm release...\n"
    helm delete $HELM_RELEASE_NAME
    exit 0
fi

printf "LOCAL_REGISTRY_RUNNING: %s\n" "$LOCAL_REGISTRY_RUNNING"
printf "KIND_CLUSTER_RUNNING: %s\n" "$KIND_CLUSTER_RUNNING"

# run local registry if not running
if [ "$LOCAL_REGISTRY_RUNNING" = "false" ]; then
    printf "Starting local registry...\n"
    docker run -d --restart=always -p "127.0.0.1:$LOCAL_REGISTRY_PORT:5000" --name "$LOCAL_REGISTRY_NAME" registry:2
    # once there is an official release of external-dns with the provider webhook, we can remove this steps
    printf "pushing external-dns-webhook-provider image to local registry...\n"
    docker pull $IMAGE_EXTERNAL_DNS_WEBHOOK_PROVIDER
    docker tag $IMAGE_EXTERNAL_DNS_WEBHOOK_PROVIDER localhost:$LOCAL_REGISTRY_PORT/external-dns-webhook-provider:latest
    docker push localhost:$LOCAL_REGISTRY_PORT/external-dns-webhook-provider:latest
fi

printf "Building binary...\n"
make build

printf "Building image...\n"
make docker-build

printf "Pushing image...\n"
make docker-push

# run kind cluster if not running
if [ "$KIND_CLUSTER_RUNNING" = "false" ]; then
    printf "Starting kind cluster...\n"
    kind create cluster  --name $KIND_CLUSTER_NAME --config $KIND_CLUSTER_CONFIG
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    sleep $KIND_CLUSTER_WAIT
    docker network connect "kind" "$LOCAL_REGISTRY_NAME"
    kubectl apply -f ./deployments/kind/local-registry-configmap.yaml
    printf "Installing dns mock server...\n"
    helm upgrade --install --namespace dns-mockserver --create-namespace dns-mockserver mockserver/mockserver -f ./deployments/dns-mockserver/dns-mockserver-values.yaml
fi

helm upgrade $HELM_RELEASE_NAME $HELM_CHART -f $HELM_VALUES_FILE --install

