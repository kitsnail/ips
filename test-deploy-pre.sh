#!/bin/bash

docker rmi ips-apiserver:latest 

make docker-build

skopeo copy \
  --override-os linux \
  --override-arch arm64 \
  --dest-tls-verify=false \
  --dest-creds admin:Harbor12345 \
  docker-daemon:ips-apiserver:latest \
  docker://cr01.home.lan/library/ips-apiserver:latest

kubectl delete -k deploy/
sleep 10

#kubectl apply -k deploy/
