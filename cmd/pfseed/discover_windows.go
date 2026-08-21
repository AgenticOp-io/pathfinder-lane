//go:build windows

package main

import (
	"net/url"

	"golang.org/x/sys/windows/registry"
)

func discoverPutty() []Candidate {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\SimonTatham\PuTTY\Sessions`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(0)
	if err != nil {
		return nil
	}
	var out []Candidate
	for _, encoded := range names {
		sk, err := registry.OpenKey(k, encoded, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		host, _, _ := sk.GetStringValue("HostName")
		user, _, _ := sk.GetStringValue("UserName")
		port, _, _ := sk.GetIntegerValue("PortNumber")
		sk.Close()
		name := encoded
		if dec, err := url.PathUnescape(encoded); err == nil && dec != "" {
			name = dec
		}
		c := Candidate{
			Name:   name,
			Host:   host,
			Port:   int(port),
			User:   user,
			Source: "putty",
		}
		if c.Host == "" {
			c.Host = name
		}
		if c.Port == 0 {
			c.Port = 22
		}
		out = append(out, c)
	}
	return out
}
