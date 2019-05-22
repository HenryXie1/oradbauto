# Automation tool to create Database 19.2 in K8S

A kubectl plugin that create statefulset of oracle database 19.2 in your
 Kubernetes cluster.

You get the full power of oracle database 19.2 in about 10-20 min and you can access it on local laptop

### Intro
Oracle Database is the foundation of many our services.  We often need to provision a Database for test, stage and prod. We would like to automate it for Oracle DB19.2 . Different versions can be added later
With this automation we can deploy a brand new Oracle DB 19.2 from your laptop via only 1 command. We can access DB from laptop within 20 min(as long as firewall ports open). We can also delete it and free resource via 1 command. It is based on DB docker images of [oracle github](https://github.com/oracle/docker-images). First time runner would take a bit more time to downloand  DB docker image which is about 6.3G
It leverages advantages of  OKE  and kubectl, we can deploy it from laptop or RUNDECK without accessing  any VMs or bastion.

### Demo
![Demo!](https://i.imgur.com/ca1MLkY.gif)

## Installation

Download kubectl via [official guide](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and configure access for your kubernetes cluster. Confirm kubectl get nodes is working
Download binary from [release link](https://github.com/HenryXie1/oradbauto/releases/download/v1.0/kubectl-oradb)
    
### Usage
```
kubectl-oradb list|create|delete [-c cdbname] [-p pdbname] [-w syspassword] [-n namespace] [flags]
Examples:
# 
#create oracle db 19c statefulset with label app=peoradbauto details in the OKE cluster
#so far we support oracle db 19.2 , new db versions can be added later
kubectl-oradb create -c cdbname -p pdbname -w syspassword 
# delete oracle db statefulset with label app=peoradbauto details in the OKE cluster. 
# Data won't be deleted.PV and PVC are kept in OKE
kubectl-oradb delete -c cdbname
# list oracle db statefulset with label app=peoradbauto details in the OKE cluster
kubectl-oradb list


Flags:
-c, --cdbname string User specified CDB name
-h, --help help for kubectl-oradb
-n, --namespace string User specified namespace (default "default")
-p, --pdbname string User specified PDB name
-w, --syspassword string sys system password of DB (default "H3YX5QRE")
Error: Please check kubectl-oradb -h for usage
```

### Contribution
More than welcome! please don't hesitate to open bugs, questions, pull requests 

 

