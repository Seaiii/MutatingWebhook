#!/bin/bash
#这个脚本为了方便调试。修改完代码执行一次即可
#删除历史镜像包
docker rmi $(docker images | grep webhook | awk '{print $3}')
#获得node的name
NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
kubectl delete -f service.yaml
kubectl delete -f deployment.yaml
kubectl delete -f mutatingWebhookConfiguration.yaml
#修改node信息可以触发mutatingWebhookConfiguration回调
kubectl label node $NODE_NAME aa-
#制作bin文件并生成镜像（makefile文件）
make image
#这个地址是你的deployment.yaml地址
DEPLOYMENT_FILE="/www/yx/webhook/deployment.yaml"
#获得最新的镜像的tag
NEW_TAG=$(docker images | grep webhook | awk '{print $2}' | sort -r | head -n 1)
#修改yaml中tag的信息
sed -i "s|image:.*|image: webhook-example:${NEW_TAG}|" ${DEPLOYMENT_FILE}
kubectl apply -f service.yaml
sleep 1
kubectl apply -f deployment.yaml
sleep 1
kubectl apply -f mutatingWebhookConfiguration.yaml
#获得pod名
POD_NAME=$(kubectl get pods -l app=webhook-example -o jsonpath='{.items[0].metadata.name}')
kubectl label node $NODE_NAME aa=bb
#可以打印日志的方式，或者自己exec进入调试
kubectl logs $POD_NAME
