#!/bin/sh

if ! type bash > /dev/null 2>&1; then
  beacon-emitter completion bash > /etc/bash_completion.d/beacon-emitter
fi

# systemctl コマンドがあるかを確認
if ! type systemctl > /dev/null 2>&1; then
  exit 0
fi

groupadd --system beacon-emitter || true
useradd --system -d /nonexistent -s /usr/sbin/nologin -g beacon-emitter beacon-emitter || true

chown beacon-emitter /etc/beacon-emitter/config.yml

systemctl daemon-reload