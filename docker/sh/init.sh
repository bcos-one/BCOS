#!/bin/bash
set -u

echo
echo "========================[init]=========================="
GETH=bcos
ClientIdentifier=bcos
echo "use ${GETH}"

echo "[*] Cleaning up temporary data directories"
rm -rf /data/${GETH}/${ClientIdentifier}
mkdir -p /data/${GETH}/logs

echo "[*] Configuring node (static)"
mkdir -p /data/${GETH}/keystore
mkdir -p /data/${GETH}/${ClientIdentifier}

if [ ! -f "/data/${GETH}/static-nodes.json" ]; then
    cp /example/static-nodes.json /data/${GETH}/
fi

if [ ! -f "/data/passwords.txt" ]; then
    touch /data/passwords.txt
fi

if [ ! -f "/data/${GETH}/${ClientIdentifier}/nodekey" ]; then
    cp /example/nodekey /data/${GETH}/${ClientIdentifier}/
fi
if [ ! -f "/data/${GETH}/keystore/key" ]; then
    cp /example/key /data/${GETH}/keystore/
fi
if [ ! -f "/data/${GETH}/genesis.json" ]; then
    cp /example/genesis.json /data/${GETH}/
fi

${GETH} --datadir /data/${GETH} init /data/${GETH}/genesis.json