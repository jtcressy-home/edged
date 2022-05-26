#!/bin/sh
systemctl stop edged.service
systemctl disable edged.service
rm /etc/systemd/system/edged.service
systemctl daemon-reload
systemctl reset-failed