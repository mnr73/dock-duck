# Runs inside the system terminal window: `docker exec -it ...`, then makes sure
# nothing is left running in the container.
#
#   sh launch.sh <docker> <context|""> <container> <marker> <user|""> <exec args...>
#
# $DOCKDUCK_CLEANUP holds cleanup.sh. Everything user-controlled arrives as
# separate argv entries (never interpolated into a command string).
docker=$1 ctx=$2 container=$3 marker=$4 user=$5
shift 5

cleanup() {
  if [ -n "$ctx" ]; then set -- --context "$ctx"; else set --; fi
  set -- "$@" exec
  if [ -n "$user" ]; then set -- "$@" -u "$user"; fi
  "$docker" "$@" "$container" sh -c "$DOCKDUCK_CLEANUP" sh "$marker" >/dev/null 2>&1
}

# closing the window hangs up the docker client; sweep after it is gone
trap 'cleanup; exit 129' HUP TERM

start=$(date +%s)
"$docker" "$@"
code=$?
cleanup

# a docker error (unknown user, container stopped, ...) must not vanish with the window
if [ "$code" -ne 0 ] && [ $(( $(date +%s) - start )) -lt 4 ]; then
  printf '\n[docker exec exited with code %s] Press Enter to close ' "$code"
  read -r _
fi
exit "$code"
