#!/bin/sh

groupadd --system beacon-emitter || true
useradd --system -d /nonexistent -s /usr/sbin/nologin -g beacon-emitter beacon-emitter || true

chown beacon-emitter /etc/beacon-emitter/config.yml

systemctl daemon-reload