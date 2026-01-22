#!/bin/bash

docker rmi ips-apiserver:latest 

make docker-build-dev

skopeo copy \
  --dest-tls-verify=false \
  --dest-creds admin:Harbor12345 \
  docker-daemon:ips-apiserver:latest \
  docker://cr01.home.lan/library/ips-apiserver:latest

kubectl apply -k deploy
kubectl rollout restart deployment ips-apiserver
