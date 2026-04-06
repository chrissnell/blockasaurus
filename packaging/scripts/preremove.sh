#!/bin/sh
set -e
systemctl stop blockasaurus.service 2>/dev/null || true
systemctl disable blockasaurus.service 2>/dev/null || true
