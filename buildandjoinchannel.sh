
#!/bin/bash -e




echo "Building channel for scmchannel" 

. setpeer.sh FirstCorp peer0
export CHANNEL_NAME="scmchannel"
peer channel create -o orderer.orderer.net:7050 -c $CHANNEL_NAME -f ./scmchannel.tx --tls true --cafile $ORDERER_CA -t 10000


. setpeer.sh FirstCorp peer0
export CHANNEL_NAME="scmchannel"
peer channel join -b $CHANNEL_NAME.block

