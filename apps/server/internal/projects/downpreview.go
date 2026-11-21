package projects

import (
	"context"

	"dockduck/internal/dockercli"
)

// DownPreview lists what `compose down` would remove, so the UI can confirm it.
type DownPreview struct {
	Project    string   `json:"project"`
	Containers []string `json:"containers"`
	Networks   []string `json:"networks"`
	// Volumes are the project's named volumes that `down -v` would delete.
	Volumes []string `json:"volumes"`
}

type nameRow struct{ Name string }

// DownPreview gathers containers (from detail), and the project-labelled
// networks and volumes. Volumes declared `external: true` carry no project
// label, matching what `down -v` leaves alone.
func (s *Service) DownPreview(ctx context.Context, ctxName string, d Detail) (DownPreview, error) {
	pv := DownPreview{Project: d.Name, Containers: []string{}, Networks: []string{}, Volumes: []string{}}
	for _, sv := range d.Services {
		for _, c := range sv.Containers {
			pv.Containers = append(pv.Containers, c.Name)
		}
	}
	if !dockercli.ValidName(d.Name) {
		return pv, nil
	}
	filter := "label=com.docker.compose.project=" + d.Name
	for _, q := range []struct {
		kind string
		dst  *[]string
	}{{"network", &pv.Networks}, {"volume", &pv.Volumes}} {
		b, err := s.Docker.Output(ctx, dockercli.Scoped(ctxName, q.kind, "ls", "--filter", filter, "--format", "json")...)
		if err != nil {
			return pv, err
		}
		rows, err := dockercli.DecodeLines[nameRow](b)
		if err != nil {
			return pv, err
		}
		for _, r := range rows {
			*q.dst = append(*q.dst, r.Name)
		}
	}
	return pv, nil
}
