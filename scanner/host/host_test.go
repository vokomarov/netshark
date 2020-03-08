package host

import "testing"

func TestHost_ID(t *testing.T) {
	for _, test := range []struct {
		IP  string
		MAC string
		ID  string
	}{
		{
			ID: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			IP: "192.168.1.1",
			ID: "66efff4c945d3c3b87fc271b47d456db",
		},
		{
			MAC: "28:6c:07:48:66:be",
			ID:  "924a7370c8df57b5b3f0a2e0d9d91598",
		},
		{
			IP:  "192.168.1.1",
			MAC: "28:6c:07:48:66:be",
			ID:  "7a48a7ba6494c7eb4c06e4787e9a6a88",
		},
	} {
		host := Host{
			IP:  test.IP,
			MAC: test.MAC,
		}

		if host.id != "" {
			t.Errorf("initial Host ID is not empty")
		}

		if host.ID() != test.ID {
			t.Errorf("test Host ID [%s] is not equal to actual Host ID [%s]", test.ID, host.ID())
		}

		if host.id == "" {
			t.Errorf("Host ID is not cached")
		}
	}
}
