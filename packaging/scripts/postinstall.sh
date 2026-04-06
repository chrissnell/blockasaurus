#!/bin/sh
set -e
systemctl daemon-reload
systemctl enable blockasaurus.service
echo "Blockasaurus installed. Edit /etc/blockasaurus/config.yml then run:"
echo "  systemctl start blockasaurus"
