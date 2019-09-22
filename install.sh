#!/bin/bash


Install(){
    LATEST_URL="https://api.github.com/repos/qinghon/BonusManger/releases/latest"

    RELEAST_INFO=$(curl -fsSL $LATEST_URL)

    TAG=$(echo $RELEAST_INFO|grep -Po '"tag_name": "\K.*?(?=")')
    DOWNLOAD_URL="https://github.com/qinghon/BonusManger/releases/download/$TAG/bonus_manger_$(uname -m)"
    mkdir -p /opt/BonusManger/bin/
    wget -O /opt/BonusManger/bin/bonusmanger $DOWNLOAD_URL
    chmod +x /opt/BonusManger/bin/bonusmanger
    cat <<EOF >/lib/systemd/system/bonus_manger.service
[Unit]
Description=bxc node app
After=network.target

[Service]
ExecStart=/opt/BonusManger/bin/bonusmanger
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable bonus_manger
    systemctl start bonus_manger    
}
remove(){
    systemctl stop bonus_manger
    systemctl disable bonus_manger
    rm -vf /lib/systemd/system/bonus_manger.service
    systemctl daemon-reload
    rm -rf /opt/BonusManger
}
case $1 in
    install|i ) Install ;;
    remove |r ) remove ;;
    * ) Install ;;
esac
shift
