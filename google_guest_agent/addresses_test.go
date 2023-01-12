//  Copyright 2017 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"

	"github.com/go-ini/ini"
)

func TestGetConfiguredVlans(t *testing.T) {
	config = ini.Empty()

	var emptySlice []string

	var tests = []struct {
		ifaces     []net.Interface
		wantIfaces []string
	}{
		{[]net.Interface{{Name: "ens3.1234"}, {Name: "ens4.5678@Google"}}, []string{"ens4.5678@Google"}},
		{[]net.Interface{}, emptySlice},
		{[]net.Interface{{Name: "ens3.1234"}, {Name: "ens4.5678"}}, emptySlice},
	}

	for _, tt := range tests {
		interfaces = tt.ifaces
		got := getConfiguredVlans()
		if !reflect.DeepEqual(got, tt.wantIfaces) {
			t.Errorf("TestGetConfiguredVlans failed: expected - %q, got - %q", tt.wantIfaces, got)
		}
	}
}

func TestAddressDisabled(t *testing.T) {
	var tests = []struct {
		name string
		data []byte
		md   *metadata
		want bool
	}{
		{"not explicitly disabled", []byte(""), &metadata{}, false},
		{"enabled in cfg only", []byte("[addressManager]\ndisable=false"), &metadata{}, false},
		{"disabled in cfg only", []byte("[addressManager]\ndisable=true"), &metadata{}, true},
		{"disabled in cfg, enabled in instance metadata", []byte("[addressManager]\ndisable=true"), &metadata{Instance: instance{Attributes: attributes{DisableAddressManager: mkptr(false)}}}, true},
		{"enabled in cfg, disabled in instance metadata", []byte("[addressManager]\ndisable=false"), &metadata{Instance: instance{Attributes: attributes{DisableAddressManager: mkptr(true)}}}, false},
		{"enabled in instance metadata only", []byte(""), &metadata{Instance: instance{Attributes: attributes{DisableAddressManager: mkptr(false)}}}, false},
		{"enabled in project metadata only", []byte(""), &metadata{Project: project{Attributes: attributes{DisableAddressManager: mkptr(false)}}}, false},
		{"disabled in instance metadata only", []byte(""), &metadata{Instance: instance{Attributes: attributes{DisableAddressManager: mkptr(true)}}}, true},
		{"enabled in instance metadata, disabled in project metadata", []byte(""), &metadata{Instance: instance{Attributes: attributes{DisableAddressManager: mkptr(false)}}, Project: project{Attributes: attributes{DisableAddressManager: mkptr(true)}}}, false},
		{"disabled in project metadata only", []byte(""), &metadata{Project: project{Attributes: attributes{DisableAddressManager: mkptr(true)}}}, true},
	}

	for _, tt := range tests {
		cfg, err := ini.InsensitiveLoad(tt.data)
		if err != nil {
			t.Errorf("test case %q: error parsing config: %v", tt.name, err)
			continue
		}
		if cfg == nil {
			cfg = &ini.File{}
		}
		newMetadata = tt.md
		config = cfg
		got := (&addressMgr{}).disabled("")
		if got != tt.want {
			t.Errorf("test case %q, addressMgr.disabled() got: %t, want: %t", tt.name, got, tt.want)
		}
	}
}

func TestAddressDiff(t *testing.T) {
	var tests = []struct {
		name string
		data []byte
		md   *metadata
		want bool
	}{
		{"not set", []byte(""), &metadata{}, false},
		{"enabled in cfg only", []byte("[wsfc]\nenable=true"), &metadata{}, true},
		{"disabled in cfg only", []byte("[wsfc]\nenable=false"), &metadata{}, false},
		{"disabled in cfg, enabled in instance metadata", []byte("[wsfc]\nenable=false"), &metadata{Instance: instance{Attributes: attributes{EnableWSFC: mkptr(true)}}}, false},
		{"enabled in cfg, disabled in instance metadata", []byte("[wsfc]\nenable=true"), &metadata{Instance: instance{Attributes: attributes{EnableWSFC: mkptr(false)}}}, true},
		{"enabled in instance metadata only", []byte(""), &metadata{Instance: instance{Attributes: attributes{EnableWSFC: mkptr(true)}}}, true},
		{"enabled in project metadata only", []byte(""), &metadata{Project: project{Attributes: attributes{EnableWSFC: mkptr(true)}}}, true},
		{"disabled in instance metadata only", []byte(""), &metadata{Instance: instance{Attributes: attributes{EnableWSFC: mkptr(false)}}}, false},
		{"enabled in instance metadata, disabled in project metadata", []byte(""), &metadata{Instance: instance{Attributes: attributes{EnableWSFC: mkptr(true)}}, Project: project{Attributes: attributes{EnableWSFC: mkptr(false)}}}, true},
		{"disabled in project metadata only", []byte(""), &metadata{Project: project{Attributes: attributes{EnableWSFC: mkptr(false)}}}, false},
	}

	for _, tt := range tests {
		cfg, err := ini.InsensitiveLoad(tt.data)
		if err != nil {
			t.Errorf("test case %q: error parsing config: %v", tt.name, err)
			continue
		}
		if cfg == nil {
			cfg = &ini.File{}
		}
		oldWSFCEnable = false
		oldMetadata = &metadata{}
		newMetadata = tt.md
		config = cfg
		got := (&addressMgr{}).diff()
		if got != tt.want {
			t.Errorf("test case %q, addresses.diff() got: %t, want: %t", tt.name, got, tt.want)
		}
	}
}

func TestWsfcFilter(t *testing.T) {
	var tests = []struct {
		metaDataJSON []byte
		expectedIps  []string
	}{
		// signle nic with enable-wsfc set to true
		{[]byte(`{"instance":{"attributes":{"enable-wsfc":"true"}, "networkInterfaces":[{"forwardedIps":["192.168.0.0", "192.168.0.1"]}]}}`), []string{}},
		// multi nic with enable-wsfc set to true
		{[]byte(`{"instance":{"attributes":{"enable-wsfc":"true"}, "networkInterfaces":[{"forwardedIps":["192.168.0.0", "192.168.0.1"]},{"forwardedIps":["192.168.0.2"]}]}}`), []string{}},
		// filter with wsfc-addrs
		{[]byte(`{"instance":{"attributes":{"wsfc-addrs":"192.168.0.1"}, "networkInterfaces":[{"forwardedIps":["192.168.0.0", "192.168.0.1"]}]}}`), []string{"192.168.0.0"}},
		// filter with both wsfc-addrs and enable-wsfc flag
		{[]byte(`{"instance":{"attributes":{"wsfc-addrs":"192.168.0.1", "enable-wsfc":"true"}, "networkInterfaces":[{"forwardedIps":["192.168.0.0", "192.168.0.1"]}]}}`), []string{"192.168.0.0"}},
		// filter with invalid wsfc-addrs
		{[]byte(`{"instance":{"attributes":{"wsfc-addrs":"192.168.0"}, "networkInterfaces":[{"forwardedIps":["192.168.0.0", "192.168.0.1"]}]}}`), []string{"192.168.0.0", "192.168.0.1"}},
	}

	config = ini.Empty()
	for idx, tt := range tests {
		var md metadata
		if err := json.Unmarshal(tt.metaDataJSON, &md); err != nil {
			t.Error("failed to unmarshal test JSON:", tt, err)
		}

		newMetadata = &md
		testAddress := addressMgr{}
		testAddress.applyWSFCFilter()

		forwardedIps := []string{}
		for _, ni := range newMetadata.Instance.NetworkInterfaces {
			forwardedIps = append(forwardedIps, ni.ForwardedIps...)
		}

		if !reflect.DeepEqual(forwardedIps, tt.expectedIps) {
			t.Errorf("wsfc filter failed test %d: expect - %q, actual - %q", idx, tt.expectedIps, forwardedIps)
		}
	}
}

func TestWsfcFlagTriggerAddressDiff(t *testing.T) {
	var tests = []struct {
		newMetadata, oldMetadata *metadata
	}{
		// trigger diff on wsfc-addrs
		{&metadata{Instance: instance{Attributes: attributes{WSFCAddresses: "192.168.0.1"}}}, &metadata{}},
		{&metadata{Project: project{Attributes: attributes{WSFCAddresses: "192.168.0.1"}}}, &metadata{}},
		{&metadata{Instance: instance{Attributes: attributes{WSFCAddresses: "192.168.0.1"}}}, &metadata{Instance: instance{Attributes: attributes{WSFCAddresses: "192.168.0.2"}}}},
		{&metadata{Project: project{Attributes: attributes{WSFCAddresses: "192.168.0.1"}}}, &metadata{Project: project{Attributes: attributes{WSFCAddresses: "192.168.0.2"}}}},
	}

	config = ini.Empty()
	for _, tt := range tests {
		oldWSFCAddresses = tt.oldMetadata.Instance.Attributes.WSFCAddresses
		newMetadata = tt.newMetadata
		oldMetadata = tt.oldMetadata
		testAddress := addressMgr{}
		if !testAddress.diff() {
			t.Errorf("old: %v new: %v doesn't trigger diff.", tt.oldMetadata, tt.newMetadata)
		}
	}
}
