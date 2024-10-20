#!/usr/bin/bash

ctx="/tmp/contexts"

kubectl config get-contexts -o name > $ctx
clusters_n=$(cat $ctx | wc -l | awk '{print $1}')
python=python3

for c in $(cat $ctx)
do
	cilium status --wait --context $c > /tmp/$c-cilium-status.log
	cilium clustermesh status --wait --context $c > /tmp/$c-cilium-clustermesh-status.log
done

for c in $(cat $ctx)
do
	cilium status -o json --context $c > /tmp/$c-cilium-status.json
	$python ./cilium-status.py /tmp/$c-cilium-status.json $clusters_n
	cilium clustermesh status -o json --context $c > /tmp/$c-cilium-clustermesh-status-connectivity.json
	$python ./cilium-clustermesh-status.py /tmp/$c-cilium-clustermesh-status-connectivity.json $clusters_n
done

#TOFIX odd number
#cat $ctx | xargs -n 2 -P 4 bash -c 'cilium connectivity test --context $0 --multi-cluster $1 | tee /tmp/$0-$1-connectivity-test.log'
#
##TOFIX odd number
cat $ctx | xargs -P 4 -I {} bash -c 'cilium sysdump --output-filename {} --context {}
