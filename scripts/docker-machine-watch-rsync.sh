#!/bin/bash
#
# The MIT License (MIT)
# Copyright © 2016 Frank Febbraro <frank@phase2technology.com>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the “Software”), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
#

set -o errexit

# @info:    Prints the usage
usage ()
{
  cat <<EOF
Usage: $0 [-m <machine-name> | -e <exclude file>] <path>

Examples:

  $ docker-machine-watch-rsync -m dev ~/Projects/supercoolthing/app

    > Watch for changes under ~/Projects/supercoolthing/app and rsync them into the Docker Machine named dev

  $ docker-machine-watch-rsync -m dev -e .devtools-watch-ignore ~/Projects/supercoolthing/app

    > Watch for changes under ~/Projects/supercoolthing/app excluding the patterns in .devtools-watch-ignore and rsync them into the Docker Machine named dev

EOF
  exit 0
}

while getopts "e:m:" opt; do
  case $opt in
    e)
      export IGNORE_FILENAME=$OPTARG
      ;;
    m)
      export DOCKER_MACHINE_NAME=$OPTARG
      ;;
    \?)
      usage
      ;;
  esac
done
shift $((OPTIND-1))

if [ "${DOCKER_MACHINE_NAME}x" ==  "x" ]; then
  echo "Machine name not specified"
  usage
fi

# Check that path was passed as an argument
[ "$#" -ge 1 ] || usage

export WATCH_PATH=$1

export DOCKER_MACHINE_IP=`docker-machine ip ${DOCKER_MACHINE_NAME}`
export DOCKER_MACHINE_SSH_URL="docker@$DOCKER_MACHINE_IP"
export DOCKER_HOST_SSH_KEY="$HOME/.docker/machine/machines/${DOCKER_MACHINE_NAME}/id_rsa"
export RSYNC_FLAGS="--links --perms --times --delete --omit-dir-times --inplace --whole-file -l"
export RSH_FLAG="--rsh=\"ssh -i $DOCKER_HOST_SSH_KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null\""
export EXCLUDES="--exclude={node_modules/,bower_components/}"

function do_rsync() {
  fsobject=$1
  fsevent=$2
  fstype=$3
  fsparent=$(dirname "${fsobject}")

  case "${fsevent}" in
    Removed)
      # Just sync the parent directory (non-recursive) to "notify" of a change
      CMD="rsync --dirs $RSYNC_FLAGS $RSH_FLAG $EXCLUDES ${fsparent}/ $DOCKER_MACHINE_SSH_URL:${fsparent}"
      ;;
    Renamed)
      # A rename launches 2 rename events, for the old and new name
      # Sync the parent directory (non-recursive) of both to "notify" of a change
      CMD="rsync --dirs $RSYNC_FLAGS $RSH_FLAG $EXCLUDES ${fsparent}/ $DOCKER_MACHINE_SSH_URL:${fsparent}"
      ;;

    *)
      CMD="rsync --recursive $RSYNC_FLAGS $RSH_FLAG $EXCLUDES ${fsobject} $DOCKER_MACHINE_SSH_URL:${fsparent}"
  esac

  CMD="$CMD 2>&1 | grep -v \"^Warning: Permanently added\""
 
  # TODO Shorten $FILE for printout. Show last 3 segments?

  echo -n "Sending changes for ${fsevent} event on ...${fsobject:30}......  "
  eval "$CMD" 2>&1
  echo "OK"
}

export -f do_rsync

if [ -f "$IGNORE_FILENAME" ]; then
  echo "Loading excludes from: ${IGNORE_FILENAME}"
  while read line
  do
    [[ -n "$line" && "$line" != [[:blank:]#]* ]] && EXCLUDE="$EXCLUDE --exclude='$line'"
  done < "$IGNORE_FILENAME"
fi

echo "Starting watch on: ${WATCH_PATH}"

WATCH_CMD="fswatch ${EXCLUDE} --print0 --event-flags --follow-links --recursive $WATCH_PATH"
eval "$WATCH_CMD" | xargs -0 -n 1 -I {} bash -c "do_rsync {}"
