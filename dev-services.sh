#!/bin/sh
# Starts and stops the KiloCenter development services that start-dev.sh runs.
#
#   dev-services.sh start NAME   start one service and print its PID
#   dev-services.sh stop         stop every service this checkout started
#   dev-services.sh log NAME     print the path of the service's log
#
# NAME is kc-core, kc-identity, kc-gateway or kc-web. A service runs through
# exec, so its pid file holds the service's own PID. Stop ends the process a
# pid file names and every process it started, and only while that process
# still runs the service's command: a stale pid file never kills anything.

KC_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
KC_PID_DIR="$KC_ROOT/../logs/pids"
KC_LOG_DIR="$KC_ROOT/../logs/runtime"
# Seconds a service gets to exit after SIGTERM before it is killed.
KC_STOP_TIMEOUT="${KC_STOP_TIMEOUT:-10}"
# The web server and the gateway stop before KC-Identity and KC-Core, so the
# gateway cannot reconnect to them while they shut down; a reconnecting
# gateway leaves port 50052 in TIME_WAIT.
KC_STOP_ORDER="kc-web kc-gateway kc-identity kc-core"

usage() {
    echo "usage: $0 start NAME | stop | log NAME  (NAME: kc-core, kc-identity, kc-gateway, kc-web)" >&2
    exit 2
}

service_dir() {
    case "$1" in
    kc-core) echo "$KC_ROOT/KC-Core" ;;
    kc-identity) echo "$KC_ROOT/KC-Identity" ;;
    kc-gateway) echo "$KC_ROOT/KC-Gateway" ;;
    kc-web) echo "$KC_ROOT/KC-Web" ;;
    *) return 1 ;;
    esac
}

# service_command is the command line a service runs as, as ps reports it.
service_command() {
    case "$1" in
    kc-core) echo "./kilocenter -config config.yaml" ;;
    kc-identity) echo "./identity -config config.yaml" ;;
    kc-gateway) echo "./gateway -config config.yaml" ;;
    kc-web) echo "bun run dev" ;;
    *) return 1 ;;
    esac
}

start_service() {
    dir=$(service_dir "$1") || usage
    mkdir -p "$KC_PID_DIR" "$KC_LOG_DIR"
    # The command is split into its words on purpose.
    (cd "$dir" && exec $(service_command "$1")) >"$KC_LOG_DIR/$1.log" 2>&1 &
    echo "$!" >"$KC_PID_DIR/$1.pid"
    echo "$!"
}

# descendants prints the PID of every process below the process $1.
descendants() {
    ps -A -o pid= -o ppid= | awk -v root="$1" '
        { kids[$2] = kids[$2] " " $1 }
        END {
            n = split(kids[root], stack, " ")
            while (n > 0) {
                pid = stack[n--]
                print pid
                m = split(kids[pid], more, " ")
                for (i = 1; i <= m; i++) stack[++n] = more[i]
            }
        }'
}

any_alive() {
    for pid in "$@"; do
        kill -0 "$pid" 2>/dev/null && return 0
    done
    return 1
}

# signal_and_wait sends the signal $1 to the processes that follow and waits
# up to KC_STOP_TIMEOUT seconds for them to exit; it fails if one outlives it.
signal_and_wait() {
    signal=$1
    shift
    kill -s "$signal" "$@" 2>/dev/null
    waited=0
    while any_alive "$@"; do
        [ "$waited" -ge "$KC_STOP_TIMEOUT" ] && return 1
        sleep 1
        waited=$((waited + 1))
    done
}

stop_service() {
    name=$1
    pidfile="$KC_PID_DIR/$name.pid"
    [ -f "$pidfile" ] || return 0
    pid=$(cat "$pidfile")
    rm -f "$pidfile"
    if [ "$(ps -o args= -p "$pid" 2>/dev/null)" != "$(service_command "$name")" ]; then
        echo "$name was not running"
        return 0
    fi
    echo "Stopping $name (PID $pid)..."
    set -- "$pid" $(descendants "$pid")
    signal_and_wait TERM "$@" || signal_and_wait KILL "$@"
    echo "$name stopped"
}

case "$1" in
start)
    [ $# -eq 2 ] || usage
    start_service "$2"
    ;;
stop)
    [ $# -eq 1 ] || usage
    for name in $KC_STOP_ORDER; do
        stop_service "$name"
    done
    ;;
log)
    [ $# -eq 2 ] && service_dir "$2" >/dev/null || usage
    echo "$KC_LOG_DIR/$2.log"
    ;;
*) usage ;;
esac
