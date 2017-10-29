#!/bin/bash
#
dir=$(dirname $0)

YEAR="$1"
MONTH="$2"
DAY="$3"

if [ "x$DAY" = x ]; then
  PREFIX="access_log.${YEAR}${MONTH}"
else 
  PREFIX="access_log.${YEAR}${MONTH}${DAY}-"
fi

if [ "x$4" = "x-l" ]; then
  ls -l  /afs/.bu.edu/cwis/logs/{software8a,software8b,software11a,software11b}/www{,2}/$YEAR/$MONTH/$PREFIX*.gz
else
  zcat /afs/.bu.edu/cwis/logs/{software8a,software8b,software11a,software11b}/www{,2}/$YEAR/$MONTH/$PREFIX*.gz | "${dir}/httplogs" 
fi
