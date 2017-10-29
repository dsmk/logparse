#!/usr/bin/python

import sys
import json
import locale
locale.setlocale(locale.LC_ALL, '')

class log_data:

  def __init__ (self):
    self._data = dict()

  def parse_file (self, fname):
    a = open(fname)
    data = json.load(a)
    a.close()
    return data

  def merge_data (self, new_data):
    data = self._data
    for key in ("OnCampus", "OnCampusBytes", "Total", "TotalBytes", "OffCampus", "OffCampusBytes" ) :
      if not data.has_key(key):
        data[key] = new_data[key]
      else:
        data[key] += new_data[key]

  def add_file (self, fname) :
    new_data = self.parse_file(fname)

    # now we need to merge the data
    self.merge_data(new_data)

  # simple helper for calculating pricing / x requests
  def ceiling (self, value):
    #print "value=%2f int=%d" % (value, int(value))
    if value - int(value) > 0:
      return int(value) + 1
    else:
      return int(value)

  def calc_cloudfront (self):
    data = self._data
    total_requests = data['OnCampus'] + data['OffCampus']
    total_kbytes = (data['OnCampusBytes'] + data['OffCampusBytes'])/(1024.0)

    # add a fixed amount for each request to bytes
    total_kbytes += (total_requests*2)

    # convert to Terrabytes
    total_tbytes = total_kbytes / (1024*1024*1024)

    # cloudfront is tiered by 10TB/40TB pricing
    price = 0.0
    if total_tbytes > 50.0 :
      total_tbytes -= 50.0
      print "  %.2f terrabytes over 50.0 adds $ %.2f / month" % (total_tbytes, total_tbytes*60)
      price += (total_tbytes*60)
      total_tbytes = 50.0
    if total_tbytes > 10.0 :
      total_tbytes -= 10.0
      print "  %.2f terrabytes over 10.0 adds $ %.2f / month" % (total_tbytes, total_tbytes*80)
      price += (total_tbytes*80)
      total_tbytes = 10.0
    print "  %.2f terrabytes under 10.0 adds $ %.2f / month" % (total_tbytes, total_tbytes*85)
    price += (total_tbytes*85)

    total_tbytes = total_kbytes / (1024*1024*1024)
    print "total bandwidth %.2f terrabytes cost= $ %.2f / month" % (total_tbytes, price)

    # now we determine how many 10k requests we have
    requests_10k = self.ceiling(total_requests / 10000.0)
    print "request cost (presuming all https which is slightly more expensive): %s requests =  $ %.2f / month" % (
        locale.format("%.2f", total_requests, grouping=True), requests_10k * 0.01)
    price += (requests_10k*0.01)

    # now we figure out the WAF costs $0.60 per million web requests
    requests_1m = self.ceiling(total_requests / 1000000.0)
    print "\nWAF per million requests: %s requests = $ %.2f / month" % (
        locale.format("%.2f", total_requests, grouping=True), requests_1m * 0.60)
    price += (requests_1m*0.60)

    print "\ntotal / month cost = $ %.2f" % (price)
    #print "requests=%s" % (locale.format("%.2f", total_requests, grouping=True))
    #print "gig of data= %s" % (total_kbytes / 1024*1024)

  def dump_data (self, label=""):
    for k, v in self._data.items():
      if k == 'Tracked':
        saved_keys = v.keys()
        print "tracked %d vhosts" % len(saved_keys)
      else:
        print "%s= %s" % (k, locale.format("%.2f", v, grouping=True))

ld = log_data()
for fname in sys.argv[1:] :
  print "adding file %s" % fname
  ld.add_file(fname)
print "\ndone loading files - now calculating results"
#ld.dump_data()
ld.calc_cloudfront()

