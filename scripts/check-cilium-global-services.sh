export KUBECONFIG=./kubeconfig.yaml
ctx = "/tmp/contexts"
kubectl config get-contexts -o name > $ctx
clusters_n=$(cat $ctx | wc -l | awk '{print $1}')

dir=k8s

sed "s/@CLUSTERS_NUMBER@/$clusters_n/g" $dir/cm.yaml.template > k8s/cm-cn.yaml.template

for c in $(cat $ctx)
do
	sed "s/@CLUSTER@/$c/g" $dir/cm-cn.yaml.template > $dir/cm-cn.yaml
	kubectl apply --context $c -f $dir/
	rm $dir/cm-cn.yaml
done

for c in $(cat $ctx)
do
	timeout 600 kubectl --context="$c" logs "$pod_name" --follow > "/tmp/logs_${c}.log" &
done

wait

for c in $(cat $ctx)
do
	if cat "/tmp/logs_${c}.log" | wc -l | awk '{print $1}' | grep -q "^${clusters_n}$"
	then
		echo "$c: ok"
	else
		echo "$c: failed"
		cat "/tmp/logs_${c}.log"
	fi
done
