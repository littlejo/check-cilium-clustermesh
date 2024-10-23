#!/usr/bin/env python3

import json
import sys

green = "\033[1;32;48m"
red = "\033[1;31;48m"

f = sys.argv[1]
cluster_number = int(sys.argv[2])

with open(f, 'r') as file:
    file_content = file.read()

def get_status(k, v):
    if k == "connected":
       connection_number = v["max"]
       if connection_number == cluster_number - 1:
          color = green
       else:
          color = red
       print(f"{color}connection_number: {connection_number}\33[0m")
    elif k == "clusters":
       for cn, status in v.items():
           cluster_name = cn
           configured_status = status["configured"]
           connected_status = status["connected"]
           if configured_status == 1 and connected_status == 1:
               color = green
           else:
               color = red
           print(f"{color}{cluster_name};{configured_status};{connected_status}\33[0m")

print("Check connectivity...")

for k, v in json.loads(file_content)['connectivity'].items():
    get_status(k, v)

print("Check kvstoremesh...")

for k, v in json.loads(file_content)['kvstoremesh']['status'].items():
    get_status(k, v)
