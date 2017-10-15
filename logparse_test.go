package main

import (
  "testing"
)

var testIPData = []map[string]string {
  { "name": "10net", "net": "10.0.0.0/8", "track": "hosts,uri" },
  { "name": "localhost", "net": "127.0.0.1/32", "track": "uri" },
}

func testIPRanges () ([]network) {
  ipranges, _ := initIPRanges(testIPData)

  return ipranges
}

func TestInitIPRanges (t *testing.T) {
  ipranges := testIPRanges()

  //t.Errorf("ipranges=%+v", ipranges)
  for num, item := range ipranges {
    t.Logf("item[%d]=%s name=%s -> %s", num, item, item.name, testIPData[num]["name"])
    if item.name != testIPData[num]["name"] {
      t.Errorf("build failed: name should be %s but is %s", testIPData[num]["name"], item.name)
    }
  }
}

func TestFindNetwork (t *testing.T) {
  ipranges := testIPRanges()

  ip, trackH, trackU, name := findNetwork(ipranges, "10.0.0.1")

  if ip != "10.0.0.1" {
    t.Errorf("test ip != 10.0.0.1")
  }
  if name != "10net" {
    t.Errorf("test range == %s, %b, %b, %s", ip, trackH, trackU, name)
  }
  //t.Errorf("Testing having a test fail %d\n", 1)
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
