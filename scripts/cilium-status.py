#!/usr/bin/env python3

import json
import sys

green = "\033[1;32;48m"
red = "\033[1;31;48m"

f = sys.argv[1]
cluster_number = int(sys.argv[2])

with open(f, 'r') as file:
    file_content = file.read()


for k, v in json.loads(file_content)['cilium_status'].items():
    connection_number = len(v["cluster-mesh"]["clusters"])
    if connection_number == cluster_number - 1:
       color = green
    else:
       color = red
    print(f"{color}connection_number: {connection_number}\33[0m")
    print(f"cluster_id;cluster_name;global_status;connected;ready;synced_ep;synced_id;synced_no;synced_sv;status")
    for el in v["cluster-mesh"]["clusters"]:
        cluster_id = el["config"]['cluster-id']
        cluster_name = el["name"]
        connected = el["connected"]
        ready = el["ready"]
        synced_ep = el["synced"]["endpoints"]
        synced_id = el["synced"]["identities"]
        synced_no = el["synced"]["nodes"]
        synced_sv = el["synced"]["services"]
        status = el["status"]
        status_bool = all([connected, ready, synced_ep, synced_no, synced_sv])
        if status_bool:
            color = green
        else:
            color = red
        
        print(f"{color}{cluster_id};{cluster_name};{status_bool};{connected};{ready};{synced_ep};{synced_id};{synced_no};{synced_sv};{status}\33[0m")

