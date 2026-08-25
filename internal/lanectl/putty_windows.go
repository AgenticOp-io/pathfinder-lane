//go:build windows

package lanectl

import (
	"net/url"

	"golang.org/x/sys/windows/registry"
)

func listPutty() []puttyEntry {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\SimonTatham\PuTTY\Sessions`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(0)
	if err != nil {
		return nil
	}
	var out []puttyEntry
	for _, encoded := range names {
		sk, err := registry.OpenKey(k, encoded, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		host, _, _ := sk.GetStringValue("HostName")
		user, _, _ := sk.GetStringValue("UserName")
		proto, _, _ := sk.GetStringValue("Protocol")
		proxyCmd, _, _ := sk.GetStringValue("ProxyTelnetCommand")
		port, _, _ := sk.GetIntegerValue("PortNumber")
		method, _, _ := sk.GetIntegerValue("ProxyMethod")
		sk.Close()
		name := encoded
		if dec, err := url.PathUnescape(encoded); err == nil && dec != "" {
			name = dec
		}
		e := puttyEntry{
			Key:         encoded,
			Name:        name,
			Host:        host,
			Port:        int(port),
			User:        user,
			Protocol:    proto,
			ProxyMethod: uint32(method),
			ProxyCmd:    proxyCmd,
		}
		if e.Host == "" {
			e.Host = name
		}
		if e.Port == 0 {
			e.Port = 22
		}
		out = append(out, e)
	}
	return out
}

func applyPutty(e puttyEntry, host string, port int, proxyMethod uint32, proxyCmd string) error {
	path := `Software\SimonTatham\PuTTY\Sessions\` + e.Key
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue("HostName", host); err != nil {
		return err
	}
	if err := k.SetDWordValue("PortNumber", uint32(port)); err != nil {
		return err
	}
	if err := k.SetDWordValue("ProxyMethod", proxyMethod); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyTelnetCommand", proxyCmd); err != nil {
		return err
	}
	return nil
}
