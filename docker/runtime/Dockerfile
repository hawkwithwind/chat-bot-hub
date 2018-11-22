FROM alpine:3.8

ARG mirror=mirrors.ustc.edu.cn
ADD modifysource.sh /home/scripts/modifysource.sh
RUN /bin/sh /home/scripts/modifysource.sh $mirror
RUN cat /etc/apk/repositories
RUN apk add --no-cache --update ca-certificates apache2-utils

RUN apk add --no-cache --update tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo 'Asia/Shanghai' > /etc/timezone

ENV LANG="zh_CN.UTF-8"
RUN echo "export LC_ALL=zh_CN.UTF-8" >> /etc/profile

