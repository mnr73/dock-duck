package projects

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"dockduck/internal/dockercli"
)

func TestServiceStatus(t *testing.T) {
	c := func(state, health string, exit int) Container {
		return Container{State: state, Health: health, ExitCode: exit}
	}
	cases := []struct {
		name string
		cs   []Container
		want string
	}{
		{"none", nil, "stopped"},
		{"running", []Container{c("running", "", 0)}, "running"},
		{"healthy", []Container{c("running", "healthy", 0)}, "running"},
		{"starting", []Container{c("running", "starting", 0)}, "starting"},
		{"unhealthy beats running", []Container{c("running", "healthy", 0), c("running", "unhealthy", 0)}, "unhealthy"},
		{"restarting", []Container{c("restarting", "", 1)}, "restarting"},
		{"exited ok", []Container{c("exited", "", 0)}, "exited"},
		{"exited error", []Container{c("exited", "", 137)}, "error"},
		{"created", []Container{c("created", "", 0)}, "stopped"},
		{"paused", []Container{c("paused", "", 0)}, "paused"},
	}
	for _, tc := range cases {
		if got, _ := serviceStatus(tc.cs); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

const cfgJSON = `{"name":"erp","services":{
 "app":{"image":"larado","ports":[{"target":80,"published":"8080","protocol":"tcp"}]},
 "queue":{"image":"larado"}}}`

const psJSON = `{"ID":"aaa111","Name":"erp-app-1","Service":"app","Image":"larado","State":"running","Status":"Up","Health":"healthy","ExitCode":0,
 "Publishers":[{"URL":"0.0.0.0","TargetPort":80,"PublishedPort":8080,"Protocol":"tcp"},{"URL":"::","TargetPort":80,"PublishedPort":8080,"Protocol":"tcp"},{"URL":"","TargetPort":9000,"PublishedPort":0,"Protocol":"tcp"}]}
{"ID":"bbb222","Name":"erp-old-1","Service":"old","Image":"x","State":"exited","Status":"Exited (1)","ExitCode":1}`

func TestDetail(t *testing.T) {
	r := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) {
		j := strings.Join(a, " ")
		switch {
		case strings.Contains(j, " ps -a"):
			return []byte(psJSON), nil
		case strings.Contains(j, " config "):
			return []byte(cfgJSON), nil
		}
		return nil, errors.New("unexpected " + j)
	})
	svc := &Service{Docker: r}
	d, err := svc.Detail(context.Background(), "default", Project{Dir: "/p", Files: []string{"/p/compose.yaml"}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "erp" || d.Total != 2 || d.Running != 1 || d.Status != "partial" {
		t.Errorf("project: %+v", d.Project)
	}
	names := []string{}
	for _, s := range d.Services {
		names = append(names, s.Name+":"+s.Status)
	}
	if !slices.Equal(names, []string{"app:running", "old:error", "queue:stopped"}) {
		t.Errorf("services: %v", names)
	}
	app := d.Services[0]
	if len(app.Ports) != 1 || app.Ports[0].HostPort != 8080 || app.Ports[0].HostIP != "0.0.0.0" {
		t.Errorf("ports (IPv6 dup and unpublished must be dropped): %+v", app.Ports)
	}
	if b, _ := json.Marshal(d.Services[2]); !strings.Contains(string(b), `"containers":[]`) {
		t.Errorf("empty containers must serialise as []: %s", b)
	}
}

func TestDetailConfigFailureStillShowsContainers(t *testing.T) {
	r := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) {
		if strings.Contains(strings.Join(a, " "), " config ") {
			return nil, errors.New("required variable FOO is missing a value")
		}
		return []byte(psJSON), nil
	})
	d, err := (&Service{Docker: r}).Detail(context.Background(), "", Project{Dir: "/p"})
	if err != nil || d.ConfigError == "" || len(d.Services) != 2 {
		t.Fatalf("%v %+v", err, d)
	}
}

func TestActionArgs(t *testing.T) {
	p := Project{Dir: "/p", Name: "erp", DockerName: true, Files: []string{"/p/a.yml", "/p/b.yml"}}
	got, err := ActionArgs("remote", p, "down", ActionOpts{Volumes: true, RemoveOrphans: true})
	if err != nil {
		t.Fatal(err)
	}
	want := "--context remote compose --project-directory /p -p erp -f /p/a.yml -f /p/b.yml --ansi never --progress plain down -v --remove-orphans"
	if strings.Join(got, " ") != want {
		t.Errorf("got  %s\nwant %s", strings.Join(got, " "), want)
	}
	up, _ := ActionArgs("", Project{Dir: "/p", Files: []string{"/p/c.yml"}}, "up", ActionOpts{Services: []string{"app"}, Build: true})
	if j := strings.Join(up, " "); strings.Contains(j, " -p ") || !strings.HasSuffix(j, "up -d --build app") {
		t.Errorf("up: %s", j)
	}
	for _, bad := range []struct {
		a string
		o ActionOpts
	}{{"rm", ActionOpts{}}, {"stop", ActionOpts{Services: []string{"a;b"}}}, {"down", ActionOpts{Services: []string{"app"}}}, {"stop", ActionOpts{Services: []string{"--all"}}}} {
		if _, err := ActionArgs("", p, bad.a, bad.o); !errors.Is(err, ErrBadAction) {
			t.Errorf("%v should be rejected: %v", bad, err)
		}
	}
}

func TestStatsCache(t *testing.T) {
	calls := 0
	r := dockercli.RunnerFunc(func(_ context.Context, a ...string) ([]byte, error) {
		calls++
		return []byte(`{"ID":"aaa111","CPUPerc":"13.64%","MemUsage":"464.7MiB / 30.81GiB","MemPerc":"1.47%"}`), nil
	})
	svc := &Service{Docker: r}
	for i := 0; i < 3; i++ {
		u, err := svc.Stats(context.Background(), "", Project{ID: "x"}, []string{"aaa111"})
		if err != nil || u["aaa111"].CPUPercent != 13.64 || u["aaa111"].MemPercent != 1.47 {
			t.Fatalf("%v %+v", err, u)
		}
	}
	if calls != 1 {
		t.Errorf("want 1 docker call, got %d", calls)
	}
	if _, err := svc.Stats(context.Background(), "", Project{ID: "y"}, []string{"--help"}); err == nil {
		t.Error("non-hex id must be rejected")
	}
}
