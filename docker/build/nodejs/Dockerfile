FROM node:8-alpine
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache --update ca-certificates apache2-utils

RUN apk add --no-cache --update tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo 'Asia/Shanghai' > /etc/timezone

ENV LANG="zh_CN.UTF-8"
RUN echo "export LC_ALL=zh_CN.UTF-8" >> /etc/profile

RUN apk add --update nodejs-npm
RUN npm config set registry=http://registry.npm.taobao.org
RUN npm install --global gulp@4.0.0

USER node

WORKDIR /home/work
