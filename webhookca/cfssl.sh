#bin/sh
#生成key，用k8s的ca证书进行签名，**注意:-ca和-ca-key=的目录是你k8s的master主机存放CA文件的位置**
cfssl gencert -ca=/etc/kubernetes/pki/ca.crt -ca-key=/etc/kubernetes/pki/ca.key -config=ca-config.json -hostname=webhook.default.svc -profile=server server-csr.json | cfssljson -bare server
#生成k8s的资源对象secret，后续需要绑定到deployment中
kubectl create secret tls admission-registry-tls  --key=server-key.pem --cert=server.pem
