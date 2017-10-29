# logparse
Simple Go log analytic program  

This program uses the ipnets.json file in the same directory to define sites and ip ranges we are interested in. 
It will output human readable output to stdout and a JSON summary to standard error.  

The helper scripts scan_*_logs.sh are BU specific in where they get the log files to scan.  I run them like:

  time ./scan_w3v_logs.sh 2017 09 2> w3v-2017-09.json | tee w3v-2017-09.log

The calc_aws_costs.py reads through all the json files on the command line and generates the AWS cost for 
CloudFront as well as WAF usage (the per million requests part but not the per ACL cost).  I run this on the
month of September 2017 by:

  ./calc_aws_costs.py *2017-09.json



