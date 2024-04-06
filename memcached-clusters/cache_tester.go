package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
)

var (
	REPLICATED = "The provided memcached cluster is replicated"
	SHARDED    = "The provided memcached cluster is sharded"
)

type arrayFlag []string

func (i *arrayFlag) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlag) Set(value string) error {
	*i = strings.Split(value, ",")
	return nil
}

func main() {
	var (
		routerPort string
		nodePorts  arrayFlag
	)
	flag.StringVar(&routerPort, "mcrouter", "11211", "port of mcrouter, defaults to 11211")
	flag.Var(&nodePorts, "memcacheds", "comma separated list of ports for memcached instances in the cluster")
	flag.Parse()

	mcrouter := NewCacheService(routerPort)

	memcacheds := make(map[string]ICacheService)

	for _, port := range nodePorts {
		memcacheds[port] = NewCacheService(port)
	}

	clusterType, err := checkIfSharded(mcrouter, memcacheds)

	if err != nil {
		log.Fatal(err)
	}

	log.Println(clusterType)
}

func checkIfSharded(router ICacheService, nodes map[string]ICacheService) (string, error) {
	err := router.Set("foo", "bar")

	if err != nil {
		return "", fmt.Errorf("problem with router: %v", err)
	}

	for port, node := range nodes {
		val, err := node.Get("foo")
		if (err != nil && errors.Is(err, memcache.ErrCacheMiss)) || (err == nil && val != "bar") {
			return SHARDED, nil
		}
		if err != nil {
			return "", fmt.Errorf("problem with node %s: %v", port, err)
		}
	}

	return REPLICATED, nil
}
