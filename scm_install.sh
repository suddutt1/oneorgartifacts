#!/bin/bash
. setpeer.sh FirstCorp peer0 
export CHANNEL_NAME="scmchannel"
peer chaincode install -n scm -v 1.0 -p github.com/scm
peer chaincode instantiate -o orderer.orderer.net:7050 --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA -C scmchannel -n scm -v 1.0 -c '{"Args":["init",""]}' -P " OR( 'FirstCorpMSP.member' ) " 
