package cmdutil_test

import (
	"reflect"
	"testing"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
)

func TestExpandAlias(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BK_CONFIG_DIR", dir)
	cfg := &config.Config{Version: 1, Aliases: map[string]string{"pl": "pr list"}}
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	streams, _, _, _ := iostreams.Test()
	factory := cmdutil.NewFactory(streams)

	args, err := cmdutil.ExpandAlias([]string{"pl", "--state", "all"}, factory)
	if err != nil {
		t.Fatalf("ExpandAlias: %v", err)
	}
	want := []string{"pr", "list", "--state", "all"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}

	args, err = cmdutil.ExpandAlias([]string{"pr", "list"}, factory)
	if err != nil {
		t.Fatalf("ExpandAlias builtin: %v", err)
	}
	if !reflect.DeepEqual(args, []string{"pr", "list"}) {
		t.Fatalf("builtin command was expanded: %v", args)
	}
}
