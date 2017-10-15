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

type trackedOverall struct {
  tracked_networks map[string]trackedData
  tracked_sites map[string]trackedData
}

type network struct {
  name string
  net *net.IPNet
  trackHosts bool
  trackURI bool
}

// use ipcalc http://jodies.de/ipcalc to test the ranges

var alreadyIP = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)
var buDomain = regexp.MustCompile(`\.bu\.edu$`)

func initIPRanges (data []map[string]string) ([]network, error) {

  ipranges := make([]network, len(data))

  for num, item := range data {
    trackHosts := false
    trackURI := false

    // set the track booleans based on the contents of the track item
    if strings.Contains(item["track"], "hosts") {
      trackHosts = true
    }
    if strings.Contains(item["track"], "uri") {
      trackURI = true
    }

    // parse the cidr into Go's internal form
    _, ipnet, err := net.ParseCIDR(item["net"])
    if err != nil {
      return nil, err
    }

    ipranges[num] = network{ item["name"], ipnet, trackHosts, trackURI } 
  }

  return ipranges, nil
}

func buildIPRanges (filename string) ([]network, error) {
  var data []map[string]string

  file, err := ioutil.ReadFile(filename)
  if err != nil {
    return nil, err
  }
  err = json.Unmarshal(file, &data)
  if err != nil {
    return nil, err
  }

  return initIPRanges (data)
}

func findNetwork (ipranges []network, ip string) (string, bool, bool, string) {
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
      return "unknownDNS", true, false, "error"
    }
  } else {
    // skip everything else
    return ip, false, false, "outsideBUDNS" 
  }

  for _, item := range ipranges {
    //fmt.Printf("item=%+v\n", item)
    if item.net.Contains(ipaddr) {
      return ipaddr.String(), item.trackHosts, item.trackURI, item.name
    }
  }

  // otherwise return our default values
  return ipaddr.String(), false, false, "default"
}


    
func trackEntry (ipranges []network, tracking map[string]trackedData, entry map[string]string ) {
  ip, trackHosts, trackURI, label := findNetwork(ipranges, entry["ip"])

  //fmt.Printf("entry(%s:%s)= %b, %s\n", entry["ip"], ip, track, label)

  //if trackHosts || trackURI {
    //DumpAccess(" parsed line", entry)

    element, isPresent := tracking[label]
    if isPresent {
      //fmt.Printf("element already present for %s\n", label)
    } else {
      host := make(map[string]int)
      base_uri := make(map[string]int)
      tracking[label] = trackedData{ 0, host, base_uri, trackHosts, trackURI }
      element = tracking[label]
    }

    element.base_uri["_total"]++
    if trackHosts {
      element.hosts[ip]++
    }
    if trackURI {
      element.base_uri[entry["base_uri"]]++
    }
  //}

  // if tracking, then record the base_uri, retcode, and https under the IP
  //
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

func dumpTracked (tracking map[string]trackedData) {
  var hostname string

  // go through the tracked info and 
  for k, v := range tracking {

    fmt.Printf("\n=======================================================================\n")
    fmt.Printf("*** %s (%d requests; %d unique hosts, %d base_uri)\n", 
      k, v.base_uri["_total"], len(v.hosts), len(v.base_uri)-1 )

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
        fmt.Printf("  %8d: %s (%s - hostname=%s)\n", item.Value, item.Key, k, hostname)
      }
    }

    if v.trackURI {
      fmt.Printf("\n * %s base_uri requests\n", k)
      tempData := sortedMap(v.base_uri)
      for _, item := range tempData {
        if item.Key != "_total" {
          fmt.Printf("  %8d: %s (%s)\n", item.Value, item.Key, k)
        }
      }
    }
  }

}

var whitespace = regexp.MustCompile(`\s+`)
//var frozen_whitespace = regexp.MustCompile(`++++`)
var quotes = regexp.MustCompile(`".*?[^\\]"`)
// get the top-level and second level names
var parseLevels = regexp.MustCompile(`^/+([^/]+)(/+)?([^/]+)?`)

func SpaceFreeze (input string) (string) {
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

  //fmt.Printf("elements=%+v\n", elements)
  //fmt.Printf("========= number=%d\n", len(elements))
  //for index := 0; index < len(elements) ; index++ {
  //  fmt.Printf("  element[%d]=(%s)\n", index, elements[index])
  //}
  request_line := SpaceThaw(elements[5])
  
  if strings.Contains(request_line, " ") {
    request_elements = whitespace.Split(request_line, -1)
  } else {
    request_elements = whitespace.Split("UNKNOWN baduri UNKNOWN", -1)
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
  //for k, v := range topLevel {
  //  fmt.Printf("%d: %s\n", k, v)
  //}
  
  if len(request_elements) > 2 {
    elen := len(request_elements[2])

    protocol = request_elements[2][0:elen-1]
  } else {
    fmt.Printf("request_line error near protocol: (%s)\n", request_line)
    fmt.Printf("  quoted=(%s)\n", quoted)
    protocol = "unknown"
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
    entry["***unknown1"] = elements[16]
  }
  if len(elements) > 17 {
    entry["***unknown2"] = elements[17]
  }
  if len(elements) > 18 {
    entry["***unknown3"] = elements[18]
  }
  if len(elements) > 19 {
    entry["virtual"] = elements[19]
  }

  return entry
}

func main() {
  ipranges, err := buildIPRanges("ipnets.json")
  if err != nil {
    log.Fatal(err)
  }
  tracking := make(map[string]trackedData)
  number := 0
  scanner := bufio.NewScanner(os.Stdin)

  for scanner.Scan() {
    line := scanner.Text()
    entry := ParseAccess(number, line)
    if entry != nil {
    trackEntry(ipranges, tracking, entry)
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

  fmt.Printf("\nTotal records: %d\n", number)
  dumpTracked(tracking)

}

