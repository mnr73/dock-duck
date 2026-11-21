# Projects and actions

Each project shows its services with state, image, published ports, CPU and memory. Actions run as background jobs; the output is kept so you can read why something failed.

| Button | What it runs |
|---|---|
| **Up** | `docker compose up -d` |
| **Start / Stop / Restart** | the same words, for the whole project, or per service from the row menu |
| **Pull** | pulls registry images. Services you build locally are skipped |
| **Build…** | builds images, with options *no cache* and *pull newer base images* |
| **Down…** | `docker compose down`. A dialog lists what will be removed first |

## Deleting volumes

`down -v` destroys named volumes, so it is never one click. In the Down dialog, tick *Also remove volumes*, read the volume names, and **type the project name** to enable the button. The "confirm before compose down" switch in Settings only skips the dialog for a plain `down`; removing volumes always goes through it.

## One action at a time

A project runs one action at once. A second one while the first runs is refused (the buttons are disabled). Different projects are independent.

## Ports

The **Ports** page lists every host port published by a running container, with its project, so you can see what holds a port before a project fails to start on it.
