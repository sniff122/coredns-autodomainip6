package autodomainip6

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"
	"unicode"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

const AUTODOMAINIP6_PLUGIN_NAME string = "autodomainip6"

type AutoDomainIP6 struct {
	Next plugin.Handler

	TTL             uint32
	AllowedPrefixes []net.IPNet
	Suffix          string
}

// ServeDNS implements the plugin.Handler interface.
func (v6 AutoDomainIP6) ServeDNS(ctx context.Context, writer dns.ResponseWriter, request *dns.Msg) (int, error) {
	if request.Question[0].Qtype != dns.TypeAAAA {
		return plugin.NextOrFailure(v6.Name(), v6.Next, ctx, writer, request)
	}

	if len(v6.AllowedPrefixes) == 0 {
		return dns.RcodeServerFailure, errors.New("No allowed prefixes configured")
	}

	responseAValue := request.Question[0].Name

	responseAValue = RemoveIP6DomainSuffix(responseAValue, v6.Suffix)

	ip := ConvertIPv6(responseAValue)
	if ip == nil {
		// Fall back to the next handler
		return plugin.NextOrFailure(v6.Name(), v6.Next, ctx, writer, request)
	}

	if !v6.isIPv6Allowed(ip) {
		// Do not process the request further, return an empty response
		message := new(dns.Msg)
		message.SetReply(request)
		writer.WriteMsg(message)
		return dns.RcodeRefused, nil
	}

	message := new(dns.Msg)
	message.SetReply(request)
	message.Authoritative = true
	message.Rcode = dns.RcodeSuccess
	hdr := dns.RR_Header{Name: request.Question[0].Name, Ttl: v6.TTL, Class: dns.ClassINET, Rrtype: dns.TypeAAAA}
	ipv6 := ConvertIPv6(responseAValue)
	message.Answer = []dns.RR{&dns.AAAA{Hdr: hdr, AAAA: ipv6}}

	writer.WriteMsg(message)

	// Return IPv6 address
	return dns.RcodeSuccess, nil
}

func RemoveDots(input string) string {
	return strings.ReplaceAll(input, ".", "")
}

func RemoveIP6DomainSuffix(input, suffix string) string {
	suffixWithDot := "." + suffix + "."
	if strings.HasSuffix(input, suffixWithDot) {
		return strings.TrimSuffix(input, suffixWithDot)
	}
	return input
}

func ConvertIPv6(ipv6 string) net.IP {
	// Remove colons and other non-hexadecimal characters
	var cleanHex []byte
	for _, r := range ipv6 {
		if unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			cleanHex = append(cleanHex, byte(r))
		}
	}

	// Ensure the string is an even length
	if len(cleanHex)%2 != 0 {
		cleanHex = append([]byte{'0'}, cleanHex...)
	}

	// Convert the modified hexadecimal string to a byte slice
	ipBytes, err := hex.DecodeString(string(cleanHex))
	if err != nil {
		return nil
	}

	// Ensure the resulting IPv6 address is 16 bytes
	if len(ipBytes) != net.IPv6len {
		return nil
	}

	// Convert the byte slice to a net.IP
	ip := net.IP(ipBytes)
	return ip
}

func (v6 AutoDomainIP6) isIPv6Allowed(ip net.IP) bool {
	for _, prefix := range v6.AllowedPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (v6 AutoDomainIP6) Name() string { return AUTODOMAINIP6_PLUGIN_NAME }

func init() {
	plugin.Register(AUTODOMAINIP6_PLUGIN_NAME, setup)
}

func setup(c *caddy.Controller) error {
	v6 := AutoDomainIP6{}
	v6.TTL = 900

	for c.Next() {
		switch c.Val() {
		case "ttl":
			possibleTTL := c.RemainingArgs()[0]
			ttl, err := strconv.ParseUint(possibleTTL, 10, 32)

			if err != nil {
				return plugin.Error(AUTODOMAINIP6_PLUGIN_NAME, err)
			} else {
				v6.TTL = uint32(ttl)
			}

		case "allowed":
			// Add IPv6 prefixes to the allowed list
			for _, arg := range c.RemainingArgs() {
				_, ipNet, err := net.ParseCIDR(arg)
				if err != nil {
					return plugin.Error(AUTODOMAINIP6_PLUGIN_NAME, err)
				}
				v6.AllowedPrefixes = append(v6.AllowedPrefixes, *ipNet)
			}

		case "suffix":
			if !c.NextArg() {
				return plugin.Error(AUTODOMAINIP6_PLUGIN_NAME, errors.New("Suffix can't be empty"))
			}
			v6.Suffix = c.Val()

		default:
			continue
		}
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		v6.Next = next
		return v6
	})

	return nil
}
