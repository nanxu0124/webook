#!/bin/bash

export ETCDCTL_API=3

# 设置etcd集群地址
ETCD_ADDR="http://localhost:12379"
YAML_FILE="etcd.yaml"

# 读取YAML文件内容
YAML_CONTENT=$(cat "$YAML_FILE")

# 将YAML内容作为字符串写入etcd
etcdctl --endpoints=$ETCD_ADDR put /config "$YAML_CONTENT"

echo "YAML配置已成功写入etcd，键为 /config"
