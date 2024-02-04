#!/bin/sh
if [ -e /etc/bash_completion.d/beacon-emitter ]; then
  rm -f /etc/bash_completion.d/beacon-emitter
fi

if [ "$1" != "remove" ]; then
	exit 0
fi

# systemctl コマンドがあるかを確認
if ! type systemctl > /dev/null 2>&1; then
  exit 0
fi

systemctl daemon-reload

userdel  beacon-emitter || true
groupdel beacon-emitter 2>/dev/null || true