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

#skopeo copy \
#  --dest-creds admin:Ck71rIzzpe1bc2gD \
 # docker-daemon:ips-apiserver:v0.0.2 \
 # docker://cr.shengshu1.hs1.paratera.com/library/ips-apiserver:v0.0.2

#skopeo copy \
#  --dest-creds admin:Ck71rIzzpe1bc2gD \
#  docker-daemon:ips-apiserver:v0.0.2 \
#  docker://cr.shengshu1.qd1.paratera.com/library/ips-apiserver:v0.0.2