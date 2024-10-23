#!/usr/bin/bash

ctx="/tmp/contexts"

kubectl config get-contexts -o name > $ctx
clusters_n=$(cat $ctx | wc -l | awk '{print $1}')
python=python3

dir=/tmp/clustermesh

for c in $(cat $ctx)
do
	mkdir -p $dir/$c
done

for c in $(cat $ctx)
do
	cilium status --wait --context $c > $dir/$c/cilium-status.log
	cilium clustermesh status --wait --context $c > $dir/$c/cilium-clustermesh-status.log
done

for c in $(cat $ctx)
do
	cilium status -o json --context $c > $dir/$c/cilium-status.json
	$python ./cilium-status.py $dir/$c/cilium-status.json $clusters_n | tee $dir/$c/cilium-status-json.log
	cilium clustermesh status -o json --context $c > $dir/$c/cilium-clustermesh-status-connectivity.json
	$python ./cilium-clustermesh-status.py $dir/$c/cilium-clustermesh-status-connectivity.json $clusters_n | tee $dir/$c/cilium-clustermesh-status-connectivity-json.log
done

cat $ctx | xargs -P $clusters_n -I {} bash -c "cilium connectivity test --context {} | tee $dir/{}/connectivity-test.log"
cat $ctx | xargs -P $clusters_n -I {} bash -c "cilium sysdump --output-filename $dir/{} --context {}"

check-cilium-clustermesh -test.v | tee $dir/terratest.log
