package main

import (
  "testing"
)

var testIPData = []map[string]string {
  { "virtual": "testdomain1", "status": "ignore" },
  { "virtual": "testdomain2", "status": "track" },
  { "virtual": "testdomain3", "status": "summarize" },
  { "site": "htbin", "status": "track" },
  { "name": "10net", "net": "10.0.0.0/8", "track": "hosts,uri" },
  { "name": "localhost", "net": "127.0.0.1/32", "track": "uri" },
}

func testIPRanges () (logConfig, error) {
  ipranges, err := initIPRanges(testIPData)

  return ipranges, err
}

func testIsOnCampus (t *testing.T, ip string, expected bool) {
  config, err := testIPRanges()
  t.Logf("config=%+v", config)
  if err != nil {
    t.Errorf("error=%+v", err)
    return
  }

  got := isOnCampus(ip)
  if got == expected {
    t.Logf("isOnCampus(%s)=%b", ip, got)
  } else {
    t.Errorf("isOnCampus(%s)=%b instead of %b", ip, got, expected)
  }
}

func TestIsOnCampus10Net (t *testing.T) {
  testIsOnCampus(t, "10.10.10.10", true)
}

func TestIsOnCampus128197 (t *testing.T) {
  testIsOnCampus(t, "128.197.20.40", true)
}

func TestIsOnCampus168122 (t *testing.T) {
  testIsOnCampus(t, "168.122.20.40", true)
}

func TestIsOnCampusOffCampus (t *testing.T) {
  testIsOnCampus(t, "100.240.100.100", false)
}

func TestInitIPRanges (t *testing.T) {
  config, err := testIPRanges()

  if err != nil {
    t.Errorf("error=%+v", err)
    return
  }

  //t.Errorf("config=%+v", config)
  for num, item := range config.ipranges {
    t.Logf("item[%d]=%s name=%s -> %s", num, item, item.name, testIPData[num]["name"])
    if item.name != testIPData[num]["name"] {
      t.Errorf("build failed: name should be %s but is %s", testIPData[num]["name"], item.name)
    }
  }
}

func TestFindNetwork (t *testing.T) {
  config, err := testIPRanges()

  if err != nil {
    t.Errorf("error=%+v", err)
    return
  }

  ip, trackH, trackU, name := findNetwork(config, "10.0.0.1")

  if ip != "10.0.0.1" {
    t.Errorf("test ip != 10.0.0.1")
  }
  if name != "10net" {
    t.Errorf("test range == %s, %b, %b, %s", ip, trackH, trackU, name)
  }
  //t.Errorf("Testing having a test fail %d\n", 1)
}

func testTrackStuff (t *testing.T, lines []string, numOnCampus int) (trackedOverall, error) {
  config, err := testIPRanges()
  if err != nil {
    t.Errorf("error=%+v", err)
    return trackedOverall{}, err
  }

  tracking := initTrackedOverall()
  number := 0
  for _, line := range lines {
    entry := ParseAccess(number, line)
    //t.Logf("entry=%+v", entry)
    if entry != nil {
      trackEntry(config, &tracking, entry)
    } else {
      t.Errorf("Error parsing line %d : %s", number, line)
    }
    number++
  }

  t.Logf("tracking=%+v", tracking)

  // check that we have the correct number of onCampus requests
  if tracking.onCampus != numOnCampus {
    t.Errorf("Incorrect number of onCampus requests afterwards (%d instead of %d)", tracking.onCampus, numOnCampus)
  }

  return tracking, nil
}

func TestHtbin10Net (t *testing.T) {
  var lines = []string {
    `10.241.26.100 - - [01/Sep/2017:00:00:08 -0400] "GET /htbin/wp-includes/js/wp-embed.min.js?ver=4.6.6 HTTP/1.1" 200 1403 0.007192 0.000000 0.000000 "http://www.bu.edu/met/programs/graduate/arts-administration/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36" 10673 + WajbSArxHDYAACmxCSUAAAVW 128.197.26.35 off:http`,
  }

  tracking, err := testTrackStuff(t, lines, 1) 

  if err != nil {
    t.Errorf("empty tracking afterwards: %+v", err)
  }

  //t.Errorf("tracking=%+v", tracking)
  //t.Errorf("tracking[_default]=%+v", tracking.tracked["_default"])
  vHostEntry, isPresent := tracking.tracked["_default"]
  if isPresent {
    //t.Errorf("isPresent: _total=%d len=%d\n", vHostEntry.networks["10net"].base_uri["_total"], len(lines))

    // double-check that we have the correct number of records in the 10net
    if vHostEntry.networks["10net"].base_uri["_total"] != len(lines) {
      t.Errorf("wrong number of entry: %d instead of %d", vHostEntry.networks["10net"].base_uri["_total"], len(lines))
    }

    // ensure that we have an htbin entry
    //t.Errorf("site=%s data=%+v", "htbin", vHostEntry.sites)
    siteEntry, sIsPresent := vHostEntry.sites["htbin"]
    //t.Logf("site: entry=%+v isPresent=%b\n", siteEntry, sIsPresent)
    if sIsPresent {
      t.Log("site htbin found: %+v", siteEntry)
    } else {
      t.Errorf("Should have entry for site htbin: %+v", vHostEntry.sites)
    }

  } else {
    t.Errorf("Did not have _default vhost")
  }
}

func TestHtbinPublic (t *testing.T) {
  var lines = []string {
    `100.241.26.100 - - [01/Sep/2017:00:00:08 -0400] "GET /htbin/wp-includes/js/wp-embed.min.js?ver=4.6.6 HTTP/1.1" 200 1403 0.007192 0.000000 0.000000 "http://www.bu.edu/met/programs/graduate/arts-administration/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36" 10673 + WajbSArxHDYAACmxCSUAAAVW 128.197.26.35 off:http`,
  }

  tracking, err := testTrackStuff(t, lines, 0) 

  if err != nil {
    t.Errorf("empty tracking afterwards: %+v", err)
  }

  //t.Errorf("tracking=%+v", tracking)
  //t.Errorf("tracking[_default]=%+v", tracking["_default"])
  vHostEntry, isPresent := tracking.tracked["_default"]
  if isPresent {
    //t.Errorf("isPresent: _total=%d len=%d\n", vHostEntry.networks["10net"].base_uri["_total"], len(lines))

    // double-check that we have the correct number of records in the 10net
    if vHostEntry.networks["10net"].base_uri["_total"] != 0 {
      t.Errorf("wrong number of entry: %d instead of %d", vHostEntry.networks["10net"].base_uri["_total"], len(lines))
    }

    // ensure that we have an htbin entry
    //t.Errorf("site=%s data=%+v", "htbin", vHostEntry.sites)
    siteEntry, sIsPresent := vHostEntry.sites["htbin"]
    //t.Logf("site: entry=%+v isPresent=%b\n", siteEntry, sIsPresent)
    if sIsPresent {
      t.Log("site htbin found: %+v", siteEntry)
    } else {
      t.Errorf("Should have entry for site htbin: %+v", vHostEntry.sites)
    }

  } else {
    t.Errorf("Did not have _default vhost")
  }
}

func TestRootPublic (t *testing.T) {
  var lines = []string {
    `100.241.26.100 - - [01/Sep/2017:00:00:08 -0400] "GET / HTTP/1.1" 200 1403 0.007192 0.000000 0.000000 "http://www.bu.edu/met/programs/graduate/arts-administration/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36" 10673 + WajbSArxHDYAACmxCSUAAAVW 128.197.26.35 off:http`,
  }

  tracking, err := testTrackStuff(t, lines, 0) 

  if err != nil {
    t.Errorf("empty tracking afterwards: %+v", err)
  }

  //t.Logf("tracking=%+v", tracking)
  //t.Errorf("tracking[_default]=%+v", tracking["_default"])
  vHostEntry, isPresent := tracking.tracked["_default"]
  if isPresent {
    //t.Errorf("isPresent: _total=%d len=%d\n", vHostEntry.networks["10net"].base_uri["_total"], len(lines))

    // double-check that we have the correct number of records in the 10net
    if vHostEntry.networks["10net"].base_uri["_total"] != 0 {
      t.Errorf("wrong number of entry: %d instead of %d", vHostEntry.networks["10net"].base_uri["_total"], len(lines))
    }

    // ensure that the sites hash is empty
    if len(vHostEntry.sites) == 0 {
      t.Log("sites map is empty")
    } else {
      t.Errorf("sites should be empty but it is %+v", vHostEntry.sites)
    }

  } else {
    t.Errorf("Did not have _default vhost")
  }
}

func TestRoot10Net (t *testing.T) {
  var lines = []string {
    `10.241.26.100 - - [01/Sep/2017:00:00:08 -0400] "GET / HTTP/1.1" 200 1403 0.007192 0.000000 0.000000 "http://www.bu.edu/met/programs/graduate/arts-administration/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36" 10673 + WajbSArxHDYAACmxCSUAAAVW 128.197.26.35 off:http`,
  }

  tracking, err := testTrackStuff(t, lines, 1) 

  if err != nil {
    t.Errorf("empty tracking afterwards: %+v", err)
  }

  //t.Logf("tracking=%+v", tracking)
  //t.Errorf("tracking[_default]=%+v", tracking["_default"])
  vHostEntry, isPresent := tracking.tracked["_default"]
  if isPresent {
    //t.Errorf("isPresent: _total=%d len=%d\n", vHostEntry.networks["10net"].base_uri["_total"], len(lines))

    // double-check that we have the correct number of records in the 10net
    if vHostEntry.networks["10net"].base_uri["_total"] != len(lines) {
      t.Errorf("wrong number of entry: %d instead of %d", vHostEntry.networks["10net"].base_uri["_total"], len(lines))
    }

    // ensure that the sites hash is empty
    if len(vHostEntry.sites) == 0 {
      t.Log("sites map is empty")
    } else {
      t.Errorf("sites should be empty but it is %+v", vHostEntry.sites)
    }

  } else {
    t.Errorf("Did not have _default vhost")
  }
}

func benchmarkParseAccess (b *testing.B, line string) {
  for n := 0; n < b.N; n++ {
    ParseAccess(1, line)
  }
}

func testParseAccess (t *testing.T, line string, expected_elapsed float64, expect map[string]string) {
  item := ParseAccess(1, line)

  for k, v := range item {
    t.Logf(" %s: (%s)", k, v)
  }

  for k, v := range expect {
    if item[k] != v {
      t.Errorf("%s: parsed (%s) instead of (%s)", k, item[k], v)
    }
  }

  // convert elapsed to a number and check it
  elapsed, err := ConvertElapsed(item["elapsed"])
  if err != nil {
    t.Error(err)
  } else {
    if elapsed != expected_elapsed {
      t.Errorf("elapsed: got (%f) expected (%f)", elapsed, expected_elapsed)
    }
  }
}

var mainTopLevel string = `67.249.231.2 - - [01/Sep/2017:00:00:08 -0400] "GET /met?ver=4.6.6 HTTP/1.1" 200 1403 0.007192 0.000000 0.000000 "http://www.bu.edu/met/programs/graduate/arts-administration/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36" 10673 + WajbSArxHDYAACmxCSUAAAVW 128.197.26.35 off:http`

func BenchmarkMainParseTopLevel (b *testing.B) {
  benchmarkParseAccess(b, mainTopLevel)
}

func TestMainParseToplevel (t *testing.T) {
  expected_elapsed := 0.007192
  expect := map[string]string {
    "ip": "67.249.231.2",
    "toplevel": "met",
    "base_uri": "/met",
    "uri": "/met?ver=4.6.6",
    "browser": `"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"`,
    "protocol": `HTTP/1.1`,
  }

  testParseAccess(t, mainTopLevel, expected_elapsed, expect)

  //t.Errorf("returned %+v", item)
}

func TestParseAccessBadRequest (t *testing.T) {
  line := `190.152.18.202 - - [12/Oct/2017:04:08:48 -0400] "u" 501 213 758:759 0.001000 0.000000 "-" "-" 4135 - DMZUxwrnCRgAABAnMH8AAADO 10.231.9.24 off:- wwwv.bu.edu -`
  expected_elapsed := 0.000759
  expect := map[string]string {
    "ip": "190.152.18.202",
    "toplevel": "-error-",
    "base_uri": "baduri",
    "uri": "baduri",
    "browser": `"-"`,
    "protocol": `UNKNOWN`,
  }

  testParseAccess(t, line, expected_elapsed, expect)

  //t.Errorf("returned %+v", item)
}

func TestMainParseOK (t *testing.T) {
  line := `67.249.231.2 - - [01/Sep/2017:00:00:08 -0400] "GET /met/wp-includes/js/wp-embed.min.js?ver=4.6.6 HTTP/1.1" 200 1403 0.007192 0.000000 0.000000 "http://www.bu.edu/met/programs/graduate/arts-administration/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36" 10673 + WajbSArxHDYAACmxCSUAAAVW 128.197.26.35 off:http`
  expected_elapsed := 0.007192
  expect := map[string]string {
    "ip": "67.249.231.2",
    "toplevel": "met",
    "base_uri": "/met/wp-includes/js/wp-embed.min.js",
    "uri": "/met/wp-includes/js/wp-embed.min.js?ver=4.6.6",
    "browser": `"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"`,
    "protocol": `HTTP/1.1`,
  }

  testParseAccess(t, line, expected_elapsed, expect)

  //t.Errorf("returned %+v", item)
}

var w3vParseOK string = `101.50.113.106 - - [12/Oct/2017:04:04:33 -0400] "GET /bubadmin/style.css?ver=1 HTTP/1.1" 200 3485 10359:10360 0.000000 0.000000 "http://blogs.bu.edu/bubadmin/contact-us/" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36" 3104 + -ZFabgrnCRgAAAwgIzYAAABD 10.231.9.24 off:http wwwv.bu.edu blogs.bu.edu`

func BenchmarkTestW3VParseOK (b *testing.B) {
  benchmarkParseAccess(b, w3vParseOK)
}

func TestW3VParseOK (t *testing.T) {
  expected_elapsed := 0.01036
  expect := map[string]string {
    "ip": "101.50.113.106",
    "toplevel" : "bubadmin",
    "base_uri": "/bubadmin/style.css",
    "uri": "/bubadmin/style.css?ver=1",
    "browser": `"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36"`,
    "protocol": `HTTP/1.1`,
    "virtual": "blogs.bu.edu",
  }

  testParseAccess(t, w3vParseOK, expected_elapsed, expect)

  //t.Errorf("returned %+v", item)
}

