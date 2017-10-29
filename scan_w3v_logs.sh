#!/bin/bash
#
dir=$(dirname $0)

ACTION=scan
LANDSCAPE=prod

if [ "x$1" = "x-c" ]; then
  ACTION=cat
  shift
fi
if [ "x$1" = "x-l" ]; then
  ACTION=list
  shift
fi
if [ "x$1" = "x-e" ]; then
  shift
  LANDSCAPE="$1"
  shift
fi
YEAR="$1"
MONTH="$2"
DAY="$3"

if [ "x$DAY" = x ]; then
  DAY='*'
fi

if [ "$ACTION" = "list" ]; then
  ls -l /archive/ServerLogs/usoftware/logs/apache/wp-w3v-${LANDSCAPE}/$YEAR/$MONTH/$DAY/access_log*gz
elif [ "$ACTION" = cat ]; then
  zcat /archive/ServerLogs/usoftware/logs/apache/wp-w3v-${LANDSCAPE}/$YEAR/$MONTH/$DAY/access_log*gz
else
  zcat /archive/ServerLogs/usoftware/logs/apache/wp-w3v-${LANDSCAPE}/$YEAR/$MONTH/$DAY/access_log*gz | "${dir}/httplogs"
fi
