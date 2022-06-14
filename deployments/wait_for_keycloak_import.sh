#!/usr/bin/env bash

COMPOSE_FILE="$1"
CONTAINER_NAME="keycloak"
COMPOSE_COMMAND=""
SUCCESS_LOG_ENTRY="Keycloak.*started in \d+ms"
START_SECONDS="$SECONDS"
TIMEOUT="60"

command_exists() {
  command -v "$1" >/dev/null
}

set_compose_command() {
  if command_exists "docker-compose"; then
    echo "docker-compose"
  elif command_exists "podman-compose"; then
    echo "podman-compose"
  fi
}

success_entry_found() {
  grep -Pq "$SUCCESS_LOG_ENTRY" <<< "$("$COMPOSE_COMMAND" -f "$COMPOSE_FILE" logs "$CONTAINER_NAME" 2>/dev/null)"
}

init_checks() {

  if ! [ -r "$COMPOSE_FILE" ]; then
    echo "cannot read compose file: '${COMPOSE_FILE}'"
    return 1
  fi

  COMPOSE_COMMAND="$(set_compose_command)"

  if [ -z "$COMPOSE_COMMAND" ]; then
    echo "cannot find either docker-compose nor podman-compose in PATH"
    return 1
  fi
}


wait_for() {

  echo -n "waiting for ${CONTAINER_NAME}"

  while ! success_entry_found; do
    echo -n '.'
    sleep 1

    if [[ $(( SECONDS - START_SECONDS )) -gt $TIMEOUT ]]; then
      "$COMPOSE_COMMAND" -f "$COMPOSE_FILE" logs "$CONTAINER_NAME"
      echo "$CONTAINER_NAME failed to reach ready status under $TIMEOUT seconds"
      return 1
    fi
  done

  echo -e "\n Took $(( SECONDS - START_SECONDS )) seconds"
}

init_checks || exit 1
wait_for || exit 1

