FROM centos:7
ADD node /node
#增加一个glog的日志目录
RUN mkdir -p /var/log/myapp
RUN chmod 777 /var/log/myapp
CMD "/node"


