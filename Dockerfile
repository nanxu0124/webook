# 基础镜像
FROM ubuntu:20.04

COPY webook /app/webook
WORKDIR /app

ENTRYPOINT ["/app/webook"]