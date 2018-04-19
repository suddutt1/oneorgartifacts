
#!/bin/bash
export ORDERER_CA=/opt/ws/crypto-config/ordererOrganizations/orderer.net/msp/tlscacerts/tlsca.orderer.net-cert.pem

if [ $# -lt 2 ];then
	echo "Usage : . setpeer.sh FirstCorp| <peerid>"
fi
export peerId=$2

if [[ $1 = "FirstCorp" ]];then
	echo "Setting to organization FirstCorp peer "$peerId
	export CORE_PEER_ADDRESS=$peerId.firstcorp.net:7051
	export CORE_PEER_LOCALMSPID=FirstCorpMSP
	export CORE_PEER_TLS_CERT_FILE=/opt/ws/crypto-config/peerOrganizations/firstcorp.net/peers/$peerId.firstcorp.net/tls/server.crt
	export CORE_PEER_TLS_KEY_FILE=/opt/ws/crypto-config/peerOrganizations/firstcorp.net/peers/$peerId.firstcorp.net/tls/server.key
	export CORE_PEER_TLS_ROOTCERT_FILE=/opt/ws/crypto-config/peerOrganizations/firstcorp.net/peers/$peerId.firstcorp.net/tls/ca.crt
	export CORE_PEER_MSPCONFIGPATH=/opt/ws/crypto-config/peerOrganizations/firstcorp.net/users/Admin@firstcorp.net/msp
fi

	