---
layout: home
hero:
  name: DockDuck
  text: Your Docker Compose projects, in a browser
  tagline: A web GUI for the projects you already have. It works on your own folders and your own compose files, one binary, nothing to re-define.
  image:
    src: /logo.svg
    alt: DockDuck
  actions:
    - theme: brand
      text: Install
      link: /guide/install
    - theme: alt
      text: Run it on a server
      link: /server/
    - theme: alt
      text: GitHub
      link: https://github.com/mnr73/dock-duck
features:
  - title: Uses your files
    details: Point it at folders. It finds the compose projects there and edits the same compose.yaml and .env you already use. No copy kept inside the tool.
  - title: Up, down, logs, shell
    details: Start, stop, restart, pull and build, live logs with search, and a real terminal inside any container, as root, UID 1000 or a user you pick.
  - title: One binary
    details: The web UI is inside the executable. Download, run, open the browser. Updating is the same one command, or a button in Settings.
  - title: Safe by default
    details: Listens on localhost only. Reaching it from the network needs a password. Deleting volumes needs you to type the project name.
---
