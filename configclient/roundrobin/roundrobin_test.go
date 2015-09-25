package roundrobin

import (
	"sort"
	"testing"
)

func TestX(t *testing.T) {
	servers := "10,www.google.com:3912;20,8.8.8.8:80;10,irl.com:1020"
	allserv := ParseServers(servers)
	sort.Sort(allserv)
	allserv.Order()
}
