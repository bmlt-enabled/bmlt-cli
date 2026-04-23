package main

import (
	"reflect"
	"testing"
)

func TestExtractCommand(t *testing.T) {
	cases := []struct {
		args     []string
		wantCmd  string
		wantArgs []string
	}{
		{[]string{"servers"}, "servers", []string{}},
		{[]string{"servers", "ohio"}, "servers", []string{"ohio"}},
		{[]string{"-s", "Ohio Region", "search", "--weekdays", "mon"}, "search", []string{"-s", "Ohio Region", "--weekdays", "mon"}},
		{[]string{"search", "-s", "Ohio Region", "--weekdays", "mon"}, "search", []string{"-s", "Ohio Region", "--weekdays", "mon"}},
		{[]string{"--server=Ohio Region", "info"}, "info", []string{"--server=Ohio Region"}},
		{[]string{"-s", "X"}, "", []string{"-s", "X"}}, // no recognized command
		{[]string{"--help"}, "--help", []string{}},
	}
	for _, tc := range cases {
		gotCmd, gotArgs := extractCommand(tc.args)
		if gotCmd != tc.wantCmd {
			t.Errorf("extractCommand(%v) cmd = %q, want %q", tc.args, gotCmd, tc.wantCmd)
		}
		if !reflect.DeepEqual(gotArgs, tc.wantArgs) {
			t.Errorf("extractCommand(%v) args = %v, want %v", tc.args, gotArgs, tc.wantArgs)
		}
	}
}

func TestSubcommandsRegistered(t *testing.T) {
	required := []string{
		"servers", "info", "search", "formats", "bodies", "service-bodies",
		"changes", "keys", "field-keys", "values", "field-values", "naws", "coverage",
	}
	for _, c := range required {
		if !subcommands[c] {
			t.Errorf("subcommand %q not registered in subcommands map", c)
		}
	}
}
