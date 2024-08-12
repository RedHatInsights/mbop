#!/usr/bin/env bash

COMPOSE_FILE="$1"
CONTAINER_NAME="keycloak"
CONTAINER_ENGINE=""
SUCCESS_LOG_ENTRY="Keycloak.*started in \d+ms"
START_SECONDS="$SECONDS"
TIMEOUT="60"

command_exists() {
  command -v "$1" >/dev/null
}

get_container_engine() {
  if command_exists "docker"; then
    echo -n "docker"
  elif command_exists "podman"; then
    echo -n "podman"
  else
    return 1
  fi
}

success_entry_found() {
  grep -Pq "$SUCCESS_LOG_ENTRY" <<<"$("$CONTAINER_ENGINE" -f "$COMPOSE_FILE" logs "$CONTAINER_NAME" 2>/dev/null)"
}

init_checks() {

  if ! [ -r "$COMPOSE_FILE" ]; then
    echo "cannot read compose file: '${COMPOSE_FILE}'"
    return 1
  fi

  if ! CONTAINER_ENGINE="$(get_container_engine)"; then
    echo "cannot find either docker-compose nor podman-compose in PATH"
    return 1
  fi
}

wait_for() {

  echo -n "waiting for ${CONTAINER_NAME}"

  while ! success_entry_found; do
    echo -n '.'
    sleep 1

    if [[ $((SECONDS - START_SECONDS)) -gt $TIMEOUT ]]; then
      "$CONTAINER_ENGINE" compose -f "$COMPOSE_FILE" logs "$CONTAINER_NAME"
      echo "$CONTAINER_NAME failed to reach ready status under $TIMEOUT seconds"
      return 1
    fi
  done

  echo -e "\n Took $((SECONDS - START_SECONDS)) seconds"
}

init_checks || exit 1
wait_for || exit 1
