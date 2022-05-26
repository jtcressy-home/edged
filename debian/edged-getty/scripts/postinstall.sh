#!/bin/sh
systemctl daemon-reload
systemctl enable edged.service
systemctl start edged.service
cat <<EOF
edged has been installed as a systemd service and has been configured to run on boot.
Additionally, edged will use tty1 to display system information.
EOF