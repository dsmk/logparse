package main

import (
  "net"
  "fmt"
  "bufio"
  "os"
  "strings"
  "regexp"
  "time"
  "log"
  "encoding/json"
  "io/ioutil"
  "sort"
  "strconv"
)

type trackedData struct {
  num_requests int
  hosts map[string]int
  base_uri map[string]int
  trackHosts bool
  trackURI bool
}

type trackedInfo struct {
  number int
  networks map[string]trackedData
  sites map[string]trackedData
}

type trackedOverall struct {
  total int
  totalBytes int64
  onCampus int
  onCampusBytes int64
  offCampus int
  offCampusBytes int64
  tracked map[string]trackedInfo
}

type network struct {
  name string
  net *net.IPNet
  trackHosts bool
  trackURI bool
  ignore bool
}

type logConfig struct {
  ipranges []network
  vhosts map[string]int
  sites map[string]int
}

// use ipcalc http://jodies.de/ipcalc to test the ranges

var alreadyIP = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
var buDomain = regexp.MustCompile(`\.bu\.edu$`)

// numberToArray (int) -> (ignoreItem, trackItems)
func numberToArray (number int) (bool, bool) {
  if number == -1 {
    return true, false
  } else if number == 0 {
    return false, false
  } else {
    return false, true
  }
}

func statusToNumber (status string) (int) {
  if status == "ignore" {
    return -1
  } else if status == "summarize" {
    return 0
  } else {
    return 1
  }
}

// we do this format for performance
var onCampusIPs = []*net.IPNet {
  &net.IPNet{ IP: net.IPv4(10,0,0,0), Mask: net.IPv4Mask(255,0,0,0) },
  &net.IPNet{ IP: net.IPv4(128,197,0,0), Mask: net.IPv4Mask(255,255,0,0) },
  &net.IPNet{ IP: net.IPv4(168,122,0,0), Mask: net.IPv4Mask(255,255,0,0) },
}

func isOnCampus (ip string) (bool) {
  ipaddr := net.ParseIP(ip)

  for _, ipnet := range onCampusIPs {
    if ipnet.Contains(ipaddr) {
      return true
    }
  }

  return false
}

func initIPRanges (data []map[string]string) (logConfig, error) {

  ipranges := make([]network, len(data))
  vhosts := make(map[string]int)
  sites := make(map[string]int)

  for num, item := range data {
    virtual, vIsPresent := item["virtual"]
    site, sIsPresent := item["site"]
    if vIsPresent {
      // we need to add the virtual host to the list
      vhosts[virtual] = statusToNumber(item["status"])

    } else if sIsPresent {
      // record the site
      sites[site] = statusToNumber(item["status"])
    } else {
      // we presume it is a network entry
      trackHosts := false
      trackURI := false

      // set the track booleans based on the contents of the track item
      if strings.Contains(item["track"], "hosts") {
        trackHosts = true
      }
      if strings.Contains(item["track"], "uri") {
        trackURI = true
      }

      // determine if we want to ignore the network (does nothing other than add to the total
      _, ignore := item["ignore"]

      // parse the cidr into Go's internal form
      _, ipnet, err := net.ParseCIDR(item["net"])
      if err != nil {
        return logConfig{}, err
      }

      ipranges[num] = network{ item["name"], ipnet, trackHosts, trackURI, ignore } 
    }

  }

  // now that we are done we need to build our structure
  return logConfig{ ipranges, vhosts, sites }, nil
}

func buildIPRanges (filename string) (logConfig, error) {
  var data []map[string]string

  file, err := ioutil.ReadFile(filename)
  if err != nil {
    return logConfig{}, err
  }
  err = json.Unmarshal(file, &data)
  if err != nil {
    return logConfig{}, err
  }

  return initIPRanges (data)
}

func findSite (config logConfig, site string) (bool, bool) {

  status, isPresent := config.sites[site]
  if isPresent {
    return numberToArray(status)
  } 

  // default is to ignore most toplevels
  return true, false
}

func findVirtual (config logConfig, vhost string) (bool, bool) {

  status, isPresent := config.vhosts[vhost]
  if isPresent {
    return numberToArray(status)
  } 

  // default is to track all virtual hosts
  return false, true
}

func findNetwork (config logConfig, ip string) (string, bool, bool, bool, string) {
  var ipaddr net.IP

  // if the ip is actually a hostname then look it up (if in bu.edu)
  if alreadyIP.MatchString(ip) {
    ipaddr = net.ParseIP(ip)
  } else if buDomain.MatchString(ip) {
    //t := time.Now()
    //fmt.Printf("%s start lookup(%s)\n", t.Format("20060102150405"), ip)
    ips, err := net.LookupIP(ip)
    //t = time.Now()
    //fmt.Printf("%s finish lookup(%s)\n", t.Format("20060102150405"), ip)
    if err == nil {
      ipaddr = ips[0]
    } else {
      //fmt.Printf("error looking up %s : %s\n", ip, err)
      return "unknownDNS", true, false, false, "error"
    }
  } else {
    // skip everything else
    return ip, false, false, false, "outsideBUDNS" 
  }

  for _, item := range config.ipranges {
    //fmt.Printf("item=%+v\n", item)
    if item.net != nil && item.net.Contains(ipaddr) {
      return ipaddr.String(), item.trackHosts, item.trackURI, item.ignore, item.name
    }
  }

  // otherwise return our default values
  return ipaddr.String(), false, false, false, "default"
}


func trackEntryItem (tracking map[string]trackedData, label string, ip string , base_uri string, trackHosts bool, trackURI bool ) {
  element, isPresent := tracking[label]
  //fmt.Printf("trackEntryItem(label=%s isPresent=%b hosts=%b uri=%b tracking=%+v\n", label, isPresent, trackHosts, trackURI, element)

  if isPresent {
    //fmt.Printf("element already present for %s\n", label)
  } else {
    tracking[label] = initTrackedData(trackHosts, trackURI)
    element = tracking[label]
  }

  element.num_requests++
  element.base_uri["_total"]++
  if trackHosts {
    element.hosts[ip]++
  }
  if trackURI {
    element.base_uri[base_uri]++
  }

  //fmt.Printf("trackEntryItem end element=%+v", element)
}

func trackEntry (config logConfig, tracking *trackedOverall, entry map[string]string ) {
  ip, trackHosts, trackURI, ignore, label := findNetwork(config, entry["ip"])

  bytes, err := convertBytes(entry["size"])
  if err != nil {
    fmt.Printf("error parsing size: %s\n", err);
  }

  // always increment the total counter
  tracking.total++
  tracking.totalBytes += bytes

  // if we are to ignore this entry then skip it
  if ignore {
    return
  }

  // determine if we are on campus or off and record the bytes and number of requests
  if isOnCampus(entry["ip"]) {
    tracking.onCampus++
    tracking.onCampusBytes += bytes
  } else {
    tracking.offCampus++
    tracking.offCampusBytes += bytes
  }

  // now we check what the virtual host wants us to do
  virtual, virtualExists := entry["virtual"]
  if ! virtualExists {
    virtual = "_default"
  }
  ignoreVHost, trackVHost := findVirtual(config, virtual)

  //fmt.Printf("virtual=%s ignore=%b track=%b\n", entry["virtual"], ignoreVHost, trackVHost)

  if ignoreVHost {
  } else {
    // first we determine which virtual host we have and get its data
    element, isPresent := tracking.tracked[virtual]
    if isPresent {
    } else {
      element = initTrackedInfo()
      tracking.tracked[virtual] = element
    }

    // if track is false then override both the trackHosts and trackURI variables
    if trackVHost {
      trackEntryItem(element.networks, label, ip, entry["base_uri"], trackHosts, trackURI)
    } else {
      trackEntryItem(element.networks, label, ip, entry["base_uri"], false, false)
    }

    // next track the toplevel if it exists
    toplevel, tExists := entry["toplevel"]
    if ! tExists {
      toplevel = "_default"
    }
  
    ignoreSite, trackSite := findSite(config, toplevel)

    //fmt.Printf("site=%s ignore=%b track=%b config=%+v\n", toplevel,  ignoreSite, trackSite, config.sites)
  
    if ignoreSite {
    } else {
      if trackSite {
        trackEntryItem(element.sites, toplevel, ip, entry["base_uri"], true, true)
      } else {
        trackEntryItem(element.sites, toplevel, ip, entry["base_uri"], false, false)
      }
    }
  }

}

type keyValue struct {
  Key string
  Value int
}

func sortedMap (data map[string]int) ([]keyValue) {
  var tempData []keyValue

  for k, v := range data {
    tempData = append(tempData, keyValue{ k, v })
  }

  sort.Slice(tempData, func(i, j int) bool { return tempData[i].Value > tempData[j].Value } )

  return tempData
}

func dumpTrackedData (label string, tracking map[string]trackedData) {
  var hostname string

  for k, v := range tracking {

    fmt.Printf("\n=======================================================================\n")
    fmt.Printf("*** %s:%s (%d requests; %d unique hosts, %d base_uri)\n", 
      label, k, v.base_uri["_total"], len(v.hosts), len(v.base_uri)-1 )

    if v.trackHosts {
      fmt.Printf("\n * %s IPs\n", k)
      tempData := sortedMap(v.hosts)
      for _, item := range tempData {
        iplist, err := net.LookupAddr(item.Key)
        if err != nil {
          hostname = fmt.Sprintf("DNS-error:%s", err)
        } else { 
          hostname = iplist[0]
        }
        fmt.Printf("  %8d: %s (%s:%s - hostname=%s)\n", item.Value, item.Key, label, k, hostname)
      }
    }

    if v.trackURI {
      fmt.Printf("\n * %s base_uri requests\n", k)
      tempData := sortedMap(v.base_uri)
      for _, item := range tempData {
        if item.Key != "_total" {
          fmt.Printf("  %8d: %s (%s:%s)\n", item.Value, item.Key, label, k)
        }
      }
    }
  }

}

func dumpTracked (tracking trackedOverall) {
  total_requests := float64(tracking.total)
  total_bytes := float64(tracking.totalBytes)
  ignored_requests := tracking.total - tracking.onCampus - tracking.offCampus
  ignored_bytes := tracking.totalBytes - tracking.onCampusBytes - tracking.offCampusBytes

  fmt.Printf("### Total requests: %d bytes=%f \n", tracking.total, total_bytes)
  fmt.Printf("### On Campus: %11d (%.2f %%) kbytes= %d (%.2f %%)\n", 
    tracking.onCampus, 100*float64(tracking.onCampus)/total_requests,
    tracking.onCampusBytes/1024, 100*float64(tracking.onCampusBytes)/total_bytes)
  fmt.Printf("### Off Campus: %11d (%.2f %%) kbytes= %d (%.2f %%)\n", 
    tracking.offCampus, 100*float64(tracking.offCampus)/total_requests,
    tracking.offCampusBytes/1024, 100*float64(tracking.offCampusBytes)/total_bytes)
  fmt.Printf("### Ignored: %11d (%.2f %%) kbytes= %d (%.2f %%)\n", 
    ignored_requests, 100*float64(ignored_requests)/total_requests,
    ignored_bytes/1024, 100*float64(ignored_bytes)/total_bytes)

  for k, v := range tracking.tracked {
    dumpTrackedData("network-"+k, v.networks)
    dumpTrackedData("sites-"+k, v.sites)
  }

}

var whitespace = regexp.MustCompile(`\s+`)
//var frozen_whitespace = regexp.MustCompile(`++++`)

var quotes = regexp.MustCompile(`"[^"]*?[^\\"]?"`)

// get the top-level and second level names
var parseLevels = regexp.MustCompile(`^/+([^/]+)?(/+)?([^/]+)?`)

func SpaceFreeze (input string) (string) {
  //fmt.Printf("SpaceFreeze: %s\n", input)
  output := whitespace.ReplaceAllLiteralString(input, "++++")
  return output
}
      
func SpaceThaw (input string) (string) {
  output := strings.Replace(input, "++++", " ", -1)
  return output
}

func DumpAccess (prefix string, entry map[string]string) {
  for k, v := range entry {
    fmt.Printf("%s[%s]=(%s)\n", prefix, k, v)
  }
}

func convertBytes (bytes_s string) (int64, error) {
  if bytes_s == "-" {
    return 0, nil
  } else {
    bytes, err := strconv.ParseInt(bytes_s, 10, 64)
    if err != nil {
      //fmt.Printf("error parsing bytes(%s): %s\n", bytes_s, err);
      return 0, err
    }
    return bytes, nil
  }
}

func ConvertElapsed (elapsed_s string) (float64, error) {
  if strings.Contains(elapsed_s, ":") {
    // two integer numbers separated by a colon - that is the time in microseconds
    elapsed_s = (strings.SplitN(elapsed_s, ":", 2))[1]
    elapsed, err := strconv.ParseFloat(elapsed_s, 64)
    if err != nil {
      return elapsed, err
    } else {
      elapsed = elapsed / 1000000
      return elapsed, nil
    }
  } else {
    // single float number - that is the time in seconds
    elapsed, err := strconv.ParseFloat(elapsed_s, 64)
    if err != nil {
      return elapsed, err
    }
    return elapsed, err
  }
}

func ParseAccess (lineno int, line string) (map[string]string) {
  var base_uri string
  var protocol string
  var uri string
  var method string
  var request_elements []string

  //fmt.Printf("%d: first (%s)\n", lineno, line)

  // first we convert whitespace inside quotes into something else
  quoted := quotes.ReplaceAllStringFunc(line, SpaceFreeze)
  elements := whitespace.Split(quoted, -1)

  if len(elements) < 17 {
    fmt.Printf("Error parsing: %s\n", quoted)
    return nil
  }

  //fmt.Printf("%d: parsed (%+v)\n", lineno, elements)

  /*
  fmt.Printf("========= number=%d\n", len(elements))
  for index := 0; index < len(elements) ; index++ {
    fmt.Printf("  element[%d]=(%s)\n", index, elements[index])
  }
  */
  request_line := SpaceThaw(elements[5])
  
  if strings.Contains(request_line, " ") {
    request_elements = whitespace.Split(request_line, -1)
  } else {
    request_elements = whitespace.Split(`"UNKNOWN baduri UNKNOWN"`, -1)
    fmt.Printf("request_line error only a garbage string: (%s)\n", request_line)
    fmt.Printf("  quoted=(%s)\n", quoted)
    for index := 0; index < len(elements) ; index++ {
      fmt.Printf("       element[%d]=(%s)\n", index, elements[index])
    }
  }

  if len(request_elements) > 0 {
    method = request_elements[0]
  } else{
    method = "(unknown)"
    fmt.Printf("request_line error near method: (%s)\n", request_line)
    fmt.Printf("  quoted=(%s)\n", quoted)
    for index := 0; index < len(elements) ; index++ {
      fmt.Printf("       element[%d]=(%s)\n", index, elements[index])
    }
  }

  if len(request_elements) > 1 {
    uri = request_elements[1]
  } else {
    fmt.Printf("request_line error near uri: (%s)\n", request_line)
    fmt.Printf("  quoted=(%s)\n", quoted)
    for index := 0; index < len(elements) ; index++ {
      fmt.Printf("       element[%d]=(%s)\n", index, elements[index])
    }
    uri = "(unknown)"
  }

  if strings.Contains(uri, "?") {
    base_uri = (strings.SplitN(uri, "?", 2))[0]
  } else {
    base_uri = request_elements[1]
  }

  // now we determine the top level and the second-level
  topLevel := parseLevels.FindStringSubmatch(base_uri)
  if len(topLevel) < 4 {
    // no match 
    topLevel = []string{ "", "-error-", "", "" }
  }
  //fmt.Printf("topLevel(%d)=%+v\n", len(topLevel), topLevel)
  /* for k, v := range topLevel {
    fmt.Printf("%d: %s\n", k, v)
  } */
  
  if len(request_elements) > 2 {
    elen := len(request_elements[2])

    protocol = request_elements[2][0:elen-1]
  } else {
    //fmt.Printf("request_line error near protocol: (%s)\n", request_line)
    //fmt.Printf("  quoted=(%s)\n", quoted)
    protocol = "HTTP/0.9"
  }

  entry := map[string]string {
    "ip": elements[0],
    "ident" : elements[1],
    "user" : elements[2],
    "date" : elements[3],
    "timezone" : elements[4],
    "request_line" : request_line,
    "method" : method,
    "uri" : uri,
    "base_uri" : base_uri,
    "toplevel": topLevel[1],
    "secondLevel": topLevel[3],
    "protocol" : protocol,
    "ret" : elements[6],
    "size" : elements[7],
    "elapsed" : elements[8],
    "cpu" : elements[9],
    "cpuchild" : elements[10],
    "referer" : SpaceThaw(elements[11]),
    "browser" : SpaceThaw(elements[12]),
    "pid" : elements[13],
    "keepalive" : elements[14],
    "uniq" : elements[15],
    //"unknown" : elements[16],
    //"https" : elements[17],
    //"virtual" : elements[18],
  }
  //fmt.Printf("done: entry=%+v\n", entry)
  //fmt.Printf("ip=%s\n", entry["ip"])

  //fmt.Printf("%d: entry %+v\n\n", lineno, entry)
  // only some lines (like w3v) will have the extra items
  if len(elements) > 16 {
    entry["serverip"] = elements[16]
  }
  if len(elements) > 17 {
    entry["https"] = elements[17]
    //if strings.Contains(elements[17], ":") {
    //  https_elements := strings.SplitN(elements[17], ":", 2)
    //  entry["https_local"] = https_elements[0]
    //  entry["https_client"] = https_elements[1]
    //}
  }
  //if len(elements) > 18 {
  //  entry["virtual_config_block"] = elements[18]
  //}
  if len(elements) > 19 {
    entry["virtual"] = elements[19]
  }

  return entry
}

func initTrackedData (trackHosts, trackURI bool) (trackedData) {
  host := make(map[string]int)
  base_uri := make(map[string]int)
  return trackedData{ 0, host, base_uri, trackHosts, trackURI }
}

func initTrackedInfo () (trackedInfo) {
  networks := make(map[string]trackedData)
  sites := make(map[string]trackedData)
  return trackedInfo{ 0, networks, sites }
}

func initTrackedOverall () (trackedOverall) {
  vhosts := make(map[string]trackedInfo)
  return trackedOverall{ tracked: vhosts }
}

func main() {
  ipranges, err := buildIPRanges("ipnets.json")
  if err != nil {
    log.Fatal(err)
  }
  //tracking := make(map[string]trackedData)
  number := 0
  scanner := bufio.NewScanner(os.Stdin)
  tracking := initTrackedOverall()

  for scanner.Scan() {
    line := scanner.Text()
    entry := ParseAccess(number, line)
    if entry != nil {
      trackEntry(ipranges, &tracking, entry)
    } else {
      fmt.Printf("%d: parse line %s\n", number, line)
    }

    if number % 100000 == 0 {
      t := time.Now()
      fmt.Printf("processed=%d (%s)\n", number, t.Format("20060102150405"))
    }
    number++
  }

  if err := scanner.Err(); err != nil {
    fmt.Fprintln(os.Stderr, "error:", err)
    os.Exit(1)
  }

  dumpTracked(tracking)

}

