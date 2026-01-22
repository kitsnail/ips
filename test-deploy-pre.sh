#!/bin/bash

docker rmi ips-apiserver:dev 

make docker-build-dev

skopeo copy \
  --dest-tls-verify=false \
  --dest-creds admin:Harbor12345 \
  docker-daemon:ips-apiserver:dev \
  docker://cr01.home.lan/library/ips-apiserver:dev

kubectl apply -k deploy
kubectl rollout restart deployment ips-apiserver
