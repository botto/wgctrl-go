package wguser

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/mdlayher/wireguardctrl/wgtypes"
)

// configureDevice configures the device at the UNIX socket specified by path.
func configureDevice(path string, cfg wgtypes.Config) error {
	c, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer c.Close()

	// Start with set command.
	var buf bytes.Buffer
	buf.WriteString("set=1\n")

	// Add any necessary configuration from cfg, then finish with an empty line.
	writeConfig(&buf, cfg)
	buf.WriteString("\n")

	// Apply configuration for the device and then check the error number.
	if _, err := io.Copy(c, &buf); err != nil {
		return err
	}

	res := make([]byte, 32)
	n, err := c.Read(res)
	if err != nil {
		return err
	}

	// errno=0 indicates success, anything else returns an error number that
	// matches definitions from errno.h.
	str := strings.TrimSpace(string(res[:n]))
	if str != "errno=0" {
		// TODO(mdlayher): return actual errno on Linux?
		return os.NewSyscallError("read", fmt.Errorf("wguser: %s", str))
	}

	return nil
}

// writeConfig writes textual configuration to w as specified by cfg.
func writeConfig(w io.Writer, cfg wgtypes.Config) {
	if cfg.PrivateKey != nil {
		fmt.Fprintf(w, "private_key=%s\n", hexKey(*cfg.PrivateKey))
	}

	if cfg.ListenPort != nil {
		fmt.Fprintf(w, "listen_port=%d\n", *cfg.ListenPort)
	}

	if cfg.FirewallMark != nil {
		fmt.Fprintf(w, "fwmark=%d\n", *cfg.FirewallMark)
	}

	if cfg.ReplacePeers {
		fmt.Fprintf(w, "replace_peers=true\n")
	}

	for _, p := range cfg.Peers {
		fmt.Fprintf(w, "public_key=%s\n", hexKey(p.PublicKey))

		if p.Remove {
			fmt.Fprintf(w, "remove=true\n")
		}

		if p.PresharedKey != nil {
			fmt.Fprintf(w, "preshared_key=%s\n", hexKey(*p.PresharedKey))
		}

		if p.Endpoint != nil {
			fmt.Fprintf(w, "endpoint=%s\n", p.Endpoint.String())
		}

		if p.PersistentKeepaliveInterval != nil {
			fmt.Fprintf(w, "persistent_keepalive_interval=%d\n", int(p.PersistentKeepaliveInterval.Seconds()))
		}

		if p.ReplaceAllowedIPs {
			fmt.Fprintf(w, "replace_allowed_ips=true\n")
		}

		for _, ip := range p.AllowedIPs {
			fmt.Fprintf(w, "allowed_ip=%s\n", ip.String())
		}
	}
}

// hexKey encodes a wgtypes.Key into a hexadecimal string.
func hexKey(k wgtypes.Key) string {
	return hex.EncodeToString(k[:])
}
