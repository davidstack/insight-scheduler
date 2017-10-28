FROM registry.iop.com:5000/os/centos:7.3.1611
MAINTAINER DamonWang<wangdk@inspur.com>
ADD insight-scheduler /insight-scheduler
RUN chmod 750 /insight-scheduler
USER root
ENTRYPOINT ["/insight-scheduler"]
