#!/usr/bin/env bash

# A simple script using curl to dump all accessible tokens of a GitLab instance
# 'glab' CLI only works in a git directory and will use the git remote url for API requests. That's why we use curl.
# listUserTokens only works with an admin GITLAB_TOKEN

set -Eo pipefail

curl --version > /dev/null || { echo "'curl' >= 7.87.0 is required"; exit 1; }
jq --version > /dev/null || { echo "'jq' is required"; exit 1; }
yq --version > /dev/null || { echo "'yq' is required"; exit 1; }

export GITLAB_TOKEN=${GITLAB_TOKEN:?"GITLAB_TOKEN is required"}
export GITLAB_HOST=${GITLAB_HOST:="https://gitlab.com"}

function __exit() {
  local msg=${1:?"msg is required"}

  echo "$msg" >&2
  exit 1
}

function __gitlabGet() {
  local url=${1:?"url must be provided"}
  # echo "GET $url" >&2

  # get all pages and return merged result
  local page=1
  local per_page=100
  local result=""
  while (true); do
    local curl_status=$(curl -sL -w "%{http_code}" -o __output.json -H "PRIVATE-TOKEN: $GITLAB_TOKEN" "$GITLAB_HOST/api/v4/$url" --url-query page=$page --url-query per_page=$per_page)
    if [ "$curl_status" -ne 200 ]; then
      echo "failed to get $url: $curl_status" >&2
      cat __output.json
      rm -f __output.json
      return 1
    fi

    result="$(echo -e "$result\n$(cat __output.json)" | jq -s 'add')"

    if [ "$(cat __output.json | jq '.|length')" -lt $per_page ]; then
      break;
    fi
    page=$((page+1))
  done
  rm -f __output.json
  if [ -n "$result" ]; then
    echo "$result"
  fi
}

# printTokens prints GitLab tokens in the 'source' format that can be used for token-operator
function printTokens() {
  local group=${1:?"group required"}
  local tokens=${2:?"tokens required"}

  # map access_level to role: https://docs.gitlab.com/api/access_requests/#valid-access-levels
  echo "$tokens" | jq -r '.[]|select(.active == true)|
    "- name: " + .name +
    "\n  state: active" +
    "\n  source:" +
    "\n    name: " + .name +
    "\n    description: " + .description +
    "\n    type: " + if has("resource_type") then .resource_type else "personal" end +
    "\n    scopes: [" + (.scopes|join(",")) + "]" +
    "\n    owner: '$group'" +
    if has("access_level") then
        "\n    role: " +
        if .access_level == 10 then "guest"
        elif .access_level == 20 then "reporter"
        elif .access_level == 30 then "developer"
        elif .access_level == 40 then "maintainer"
        elif .access_level == 50 then "owner"
        else .access_level|tostring
        end
    else ""
    end +
    "\n    id: " + (.id|tostring) +
    "\n    expires_at: " + .expires_at +
    "\n    last_used_at: " + .last_used_at +
    "\n  vault: {}"
    ' | yq
}

# loops over all active groups and their subgroups and prints access tokens
function listGroupTokens() {
  groups=$(__gitlabGet "groups?active=true")
  [ $? -ne 0 ] && return 1

  for id in $(echo "$groups" | jq -r '.[]|.id') ; do
    gpath=$(echo "$groups" | jq -r '.[]|select(.id == '$id')|.full_path')

    echo "Group: $gpath ($id)" >&2

    tokens=$(__gitlabGet "groups/$id/access_tokens?state=active")
    [ $? -ne 0 ] && return 1

    if [ "$tokens" != "null" ] && [ $(echo "$tokens" | jq '[.[]|select(.active == true)]|length') -gt 0 ]; then
      printTokens "$gpath" "$tokens"
    fi

    subgroups=$(__gitlabGet "groups/$id/subgroups?active=true")
    [ $? -ne 0 ] && return 1

    for sid in $(echo "$subgroups" | jq -r '.[]|.id'); do
      [ -z "$sid" ] && continue
      [ -n "$(echo "$groups" | jq -r '.[]|.id' | grep "^$sid$")" ] && continue
      spath=$(echo "$subgroups" | jq -r '.[]|select(.id == '$sid')|.full_path')

      echo "Group: $spath ($sid)" >&2

      tokens=$(__gitlabGet "groups/$sid/access_tokens?state=active")
      [ $? -ne 0 ] && return 1

      if [ "$tokens" != "null" ] && [ $(echo "$tokens" | jq '.|length') -gt 0 ]; then
        printTokens "$spath" "$tokens"
      fi
    done
  done
}

# loops over all active projects and prints access tokens
function listProjectTokens() {
  projects=$(__gitlabGet "projects?active=true")
  [ $? -ne 0 ] && return 1

  for id in $(echo "$projects" | jq -r '.[]|.id'); do
    name=$(echo "$projects" | jq -r '.[]|select(.id == '$id')|.name_with_namespace')

    echo "Project: $name ($id)" >&2

    tokens=$(__gitlabGet "projects/$id/access_tokens?state=active")
    [ $? -ne 0 ] && return 1

    if [ "$tokens" != "" ] && [ $(echo "$tokens" | jq '.|length') -gt 0 ]; then
      printTokens "$name" "$tokens"
    fi
  done
}

# loops over all active users and prints personal access tokens
function listUserTokens() {
  users=$(__gitlabGet "users?active=true")
  [ $? -ne 0 ] && return 1

  for id in $(echo "$users" | jq -r '.[]|.id'); do
    name=$(echo "$users" | jq -r '.[]|select(.id == '$id')|.username')

    echo "User: $name ($id)" >&2

    tokens=$(__gitlabGet "personal_access_tokens?user_id=$id&state=active")
    [ $? -ne 0 ] && return 1

    if [ "$tokens" != "" ] && [ "$tokens" != "null" ] && [ $(echo "$tokens" | jq '.|length') -gt 0 ]; then
      printTokens "$name" "$tokens"
    fi
  done
}

echo "Using $GITLAB_HOST (API: $GITLAB_HOST/api/v4)" >&2

echo "tokens:"
listGroupTokens || __exit "list group tokens failed"
listProjectTokens || __exit "list project tokens failed"
listUserTokens || __exit "list user tokens failed"
