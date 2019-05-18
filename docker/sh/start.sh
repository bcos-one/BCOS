#!/bin/bash
set -u

echo
echo "========================[start]=========================="
GETH=bcos
echo "Use ${GETH}"

NETWORK_ID=2019
mkdir -p /data/${GETH}/logs

echo "[*] Starting ${GETH} nodes with ChainID and NetworkId of $NETWORK_ID"
echo
echo "Node configured. See '/data/${GETH}/logs' for logs, and run e.g. 'geth attach /data/${GETH}/${GETH}.ipc' to attach to the first ${GETH} node."
${GETH} --datadir /data/${GETH} --txpool.nolocals --nodiscover --networkid $NETWORK_ID --syncmode full --mine --minerthreads 1 --rpc --rpcaddr 0.0.0.0 --rpcapi db,eth,debug,miner,net,shh,txpool,personal,web3,${GETH} --rpccorsdomain='*' --verbosity 3 --rpcport 9545 --port 30303 --unlock 0 --password /data/passwords.txt