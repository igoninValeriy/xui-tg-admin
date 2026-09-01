package helpers

import "testing"

func TestCallbackCommand(t *testing.T) {
	cases := map[string]string{
		"\fnode_status_refresh":        "node_status_refresh",
		"\fnode_status_refresh|member": "node_status_refresh",
		"node_status_refresh":          "node_status_refresh",
		"":                             "",
	}

	for data, want := range cases {
		if got := CallbackCommand(data); got != want {
			t.Errorf("CallbackCommand(%q) = %q, want %q", data, got, want)
		}
	}
}
