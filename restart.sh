#!/bin/bash
if [ 0"$SERVERMODE" = "0" ]; then 
  export SERVERMODE=test
fi 

BIN=`pwd`/option-kline
LOGDIR=`pwd`/logs/

if [ $SERVERMODE = "test" ]; then
  BIN=/data/apps/option-kline/option-kline
  LOGDIR=/data/logs/option-kline/
elif [ $SERVERMODE = "pre_online" ]; then
  BIN=/data/apps/pre-option-kline/option-kline
  LOGDIR=/data/logs/pre-option-kline-releases/
elif [ $SERVERMODE = "online" ]; then
  BIN=/data/apps/option-kline/option-kline
  LOGDIR=/data/logs/option-kline-releases/
fi 
DIR=`dirname $BIN`
export GIN_MODE=release

if [ ! -d "$LOGDIR" ]; then
    mkdir $LOGDIR
fi

ID=$(/usr/sbin/pidof "$BIN")
if [ "$ID" ] ; then
  echo "kill  $ID"
  kill  $ID
fi

while :
do
  ID=$(/usr/sbin/pidof "$BIN")
  if [ "$ID" ] ; then
    echo "option-kline is running..."
    kill $ID
    sleep 0.2
  else
    echo "option-kline service is not running"
    echo "begin to start option-kline service..."
    
    ulimit -c unlimited
   #nohup $BIN >$DIR/stdout.txt 2>&1 & 
    nohup $BIN >/dev/null 2>&1 & 
    echo "success to start option-kline service "
    break
  fi
done
