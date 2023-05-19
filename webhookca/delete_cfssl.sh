#bin/sh
rm -f server.csr
rm -f server-key.pem
rm -f server.pem
kubectl delete secret admission-registry-tls
