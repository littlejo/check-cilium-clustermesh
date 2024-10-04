#!/usr/bin/bash

export KUBECONFIG=./kubeconfig.yaml

kubectl config get-contexts -o name > /tmp/contexts

for c in $(cat /tmp/contexts)
do
        cilium status -o json --context $c | jq '.cilium_status[]."cluster-mesh".clusters' > /tmp/$c-cilium-status.json
        cilium clustermesh status -o json --context $c | jq .kvstoremesh > /tmp/$c-cilium-clustermesh-status-kvstoremesh.json
        cilium clustermesh status -o json --context $c | jq .connectivity > /tmp/$c-cilium-clustermesh-status-connectivity.json
#TOFIX Analyze this file
done

#TOFIX odd number
cat /tmp/contexts | xargs -n 2 -P 4 bash -c 'cilium connectivity test --context $0 --multi-cluster $1 | tee /tmp/$0-$1-connectivity-test.log'

for c in $(cat /tmp/contexts)
do
        cilium sysdump --output-filename $c --context $c
done
