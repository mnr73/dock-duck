# Ends every process of one terminal session inside a container.
# $1 = session marker (value of DOCKDUCK_TERM). Needs only POSIX sh + /proc.
# Must run as the session's own user: reading another user's /proc/PID/environ needs
# CAP_SYS_PTRACE, which Docker drops by default (even for root).
M="$1"
sig() {
  for p in /proc/[0-9]*; do
    pid=${p#/proc/}
    [ "$pid" = "$$" ] && continue
    if tr '\0' '\n' < "$p/environ" 2>/dev/null | grep -qx "DOCKDUCK_TERM=$M"; then
      kill -"$1" "$pid" 2>/dev/null
    fi
  done
}
sig HUP
sleep 1
sig KILL
