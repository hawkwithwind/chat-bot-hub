FROM alpine:3.8
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache --update ca-certificates apache2-utils

RUN apk add --no-cache --update tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo 'Asia/Shanghai' > /etc/timezone

ENV LANG="zh_CN.UTF-8"
RUN echo "export LC_ALL=zh_CN.UTF-8" >> /etc/profile

