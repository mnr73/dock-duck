package ports

import (
	"context"
	"strings"
	"testing"

	"dockduck/internal/dockercli"
)

const inspectJSON = `[
 {"Id":"aaaaaaaaaaaa1111","Name":"/erp-laravel","Config":{"Image":"larado","Labels":{"com.docker.compose.project":"erp","com.docker.compose.service":"laravel"}},
  "NetworkSettings":{"Ports":{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"80"},{"HostIp":"::","HostPort":"80"}],"9000/tcp":null,"8081/tcp":[{"HostIp":"127.0.0.1","HostPort":"8081"}]}}},
 {"Id":"bbbbbbbbbbbb2222","Name":"/plain","Config":{"Image":"redis","Labels":{}},
  "NetworkSettings":{"Ports":{"6379/tcp":[{"HostIp":"","HostPort":"6379"}],"53/udp":[{"HostIp":"0.0.0.0","HostPort":"5353"}]}}}
]`

func TestList(t *testing.T) {
	var calls []string
	r := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) {
		calls = append(calls, strings.Join(a, " "))
		if a[len(a)-1] == "--no-trunc" {
			return []byte("aaaaaaaaaaaa1111\nbbbbbbbbbbbb2222\n"), nil
		}
		return []byte(inspectJSON), nil
	})
	got, err := (&Service{Docker: r}).List(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 entries (80 merged, 9000 unpublished dropped), got %+v", got)
	}
	// sorted by host port: 80, 5353, 6379, 8081
	ports := []int{got[0].HostPort, got[1].HostPort, got[2].HostPort, got[3].HostPort}
	if ports[0] != 80 || ports[1] != 5353 || ports[2] != 6379 || ports[3] != 8081 {
		t.Errorf("order %v", ports)
	}
	if len(got[0].HostIPs) != 2 || got[0].Project != "erp" || got[0].Service != "laravel" || got[0].Container != "erp-laravel" || got[0].ContainerID != "aaaaaaaaaaaa" {
		t.Errorf("merged v4+v6 entry: %+v", got[0])
	}
	if got[1].Proto != "udp" || got[1].Project != "" {
		t.Errorf("udp / non-compose: %+v", got[1])
	}
	if got[2].HostIPs[0] != "0.0.0.0" {
		t.Errorf("empty host ip must mean all interfaces: %+v", got[2])
	}
	if !strings.HasPrefix(calls[0], "--context default ps") {
		t.Errorf("calls: %v", calls)
	}
}

func TestListEmptyAndBadID(t *testing.T) {
	empty := dockercli.RunnerFunc(func(context.Context, ...string) ([]byte, error) { return nil, nil })
	got, err := (&Service{Docker: empty}).List(context.Background(), "")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
	bad := dockercli.RunnerFunc(func(context.Context, ...string) ([]byte, error) { return []byte("--all\n"), nil })
	if _, err := (&Service{Docker: bad}).List(context.Background(), ""); err == nil {
		t.Fatal("non-hex id must be rejected")
	}
}
