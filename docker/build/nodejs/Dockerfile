FROM node:8-stretch-slim
ARG mirror=mirrors.ustc.edu.cn
ADD modifysource.sh /home/scripts/modifysource.sh
RUN /bin/bash /home/scripts/modifysource.sh $mirror
RUN cat /etc/apt/sources.list
RUN apt-get update

ENV TZ=Asia/Shanghai
RUN apt-get install -y tzdata
RUN rm /etc/localtime && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone

RUN apt-get install -y libssl-dev libz-dev
RUN apt-get install -y openssl

RUN apt-get install -y locales && \
    locale-gen && \
    sed -i 's/# zh_CN.UTF-8 UTF-8/zh_CN.UTF-8 UTF-8/g' /etc/locale.gen && \
    locale-gen && update-locale
ENV LANG="zh_CN.UTF-8"
RUN locale -a

#RUN apt-get install -y nodejs-npm
RUN npm config set registry=http://registry.npm.taobao.org -g
RUN npm install --global gulp@4.0.0

WORKDIR /home/work
