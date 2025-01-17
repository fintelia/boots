package main

import (
	"flag"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tinkerbell/ipxedust"
)

func TestParser(t *testing.T) {
	want := &config{
		ipxe: ipxedust.Command{
			TFTPAddr:             "0.0.0.0:69",
			TFTPTimeout:          time.Second * 5,
			EnableTFTPSinglePort: false,
		},
		ipxeTFTPEnabled:    true,
		ipxeHTTPEnabled:    true,
		ipxeRemoteTFTPAddr: "192.168.2.225",
		ipxeRemoteHTTPAddr: "192.168.2.225:8080",
		httpAddr:           "192.168.2.225:8080",
		dhcpAddr:           "0.0.0.0:67",
		syslogAddr:         "0.0.0.0:514",
		logLevel:           "info",
	}
	got := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	args := []string{
		"-log-level", "info",
		"-ipxe-remote-tftp-addr", "192.168.2.225",
		"-ipxe-remote-http-addr", "192.168.2.225:8080",
		"-http-addr", "192.168.2.225:8080",
		"-dhcp-addr", "0.0.0.0:67",
		"-syslog-addr", "0.0.0.0:514",
	}
	cli := newCLI(got, fs)
	cli.Parse(args)
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(ipxedust.Command{}, "Log"), cmp.AllowUnexported(config{})); diff != "" {
		t.Fatal(diff)
	}
}

func TestParseDynamicIPXEVarsFunc(t *testing.T) {
	tests := []struct {
		name     string
		ipxevars string
		want     [][]string
		wantErr  bool
	}{
		{
			name:     "Empty string input",
			ipxevars: "",
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "Single var definition",
			ipxevars: "myvar1=myval1",
			want:     [][]string{{"myvar1", "myval1"}},
			wantErr:  false,
		},
		{
			name:     "Two var definitions",
			ipxevars: "myvar1=myval1 myvar2=myval2",
			want:     [][]string{{"myvar1", "myval1"}, {"myvar2", "myval2"}},
			wantErr:  false,
		},
		{
			name:     "Single quotes in var definition",
			ipxevars: "'myvar1'='myval1'",
			want:     [][]string{{"'myvar1'", "'myval1'"}},
			wantErr:  false,
		},
		{
			name:     "Double quotes in var definition",
			ipxevars: "\"myvar1\"=\"myval1\"",
			want:     [][]string{{"\"myvar1\"", "\"myval1\""}},
			wantErr:  false,
		},
		{
			name:     "Invalid var definition - no equals specified",
			ipxevars: "abcdefg",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "Invalid var definition - spaces inside varname",
			ipxevars: "my var one=myval1",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "Invalid var definition - spaces inside value",
			ipxevars: "myvar1=my val one",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "Invalid var definition - just passing '='",
			ipxevars: "=",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDynamicIPXEVars(tt.ipxevars)
			if tt.wantErr {
				if err != nil {
					// pass
					return
				}
				t.Fatalf("parseDynamicIPXEVars() did not return an error, instead returned %v", got)
			}
			if err != nil {
				t.Fatalf("parseDynamicIPXEVars() returned an unexpected error: %s", err)
			}

			want := tt.want
			if !cmp.Equal(want, got) {
				t.Fatalf("parseDynamicIPXEVars() mismatch, want %v, got %v", want, got)
			}
		})
	}
}

func TestCustomUsageFunc(t *testing.T) {
	var defaultIP net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ip.IP.To4()
		if v4 == nil || !v4.IsGlobalUnicast() {
			continue
		}
		defaultIP = v4

		break
	}

	want := fmt.Sprintf(`USAGE
  Run Boots server for provisioning

FLAGS
  -dhcp-addr              IP and port to listen on for DHCP. (default "%v:67")
  -extra-kernel-args      Extra set of kernel args (k=v k=v) that are appended to the kernel cmdline when booting via iPXE.
  -http-addr              local IP and port to listen on for the serving iPXE binaries and files via HTTP. (default "%[1]v:80")
  -ipxe-enable-http       enable serving iPXE binaries via HTTP. (default "true")
  -ipxe-enable-tftp       enable serving iPXE binaries via TFTP. (default "true")
  -ipxe-remote-http-addr  remote IP and port where iPXE binaries are served via HTTP. Overrides -http-addr for iPXE binaries only.
  -ipxe-remote-tftp-addr  remote IP where iPXE binaries are served via TFTP. Overrides -tftp-addr.
  -ipxe-tftp-addr         local IP and port to listen on for serving iPXE binaries via TFTP (port must be 69). (default "0.0.0.0:69")
  -ipxe-tftp-timeout      local iPXE TFTP server requests timeout. (default "5s")
  -ipxe-vars              additional variable definitions to include in all iPXE installer scripts. Separate multiple var definitions with spaces, e.g. 'var1=val1 var2=val2'.
  -kube-namespace         An optional Kubernetes namespace override to query hardware data from.
  -kubeconfig             The Kubernetes config file location. Only applies if DATA_MODEL_VERSION=kubernetes.
  -kubernetes             The Kubernetes API URL, used for in-cluster client construction. Only applies if DATA_MODEL_VERSION=kubernetes.
  -log-level              log level. (default "info")
  -osie-path-override     A custom URL for OSIE/Hook images.
  -syslog-addr            IP and port to listen on for syslog messages. (default "%[1]v:514")
`, defaultIP)
	c := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cli := newCLI(c, fs)
	got := customUsageFunc(cli)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatal(diff)
	}
}
