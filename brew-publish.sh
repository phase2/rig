#!/usr/bin/env bash
#
# Publish a release of Rig.
#
# This entails
#  1) Creating an archive of the correct files
#  2) Getting a checksum of the archive
#  3) Publishing the archive to S3
#  4) Generating a Homebrew formula for the release
#
# Once completed, the only thing that remains is committing/pushing the
# changes in the homebrew-outrigger repo.
#

BASE_DIR=${PWD}
BIN_DIR="${BASE_DIR}/build/darwin"
VERSION=`${BIN_DIR}/rig --version | awk '{print $3}'`

FILENAME="rig-${VERSION}.tar.gz"
DIST_DIR="${BASE_DIR}/dist"
DIST="${DIST_DIR}/${FILENAME}"
S3_BUCKET="phase2.outrigger"

HOMEBREW_DIR="${HOME}/Projects/homebrew-outrigger"
HOMEBREW_FORMULA="${HOMEBREW_DIR}/Formula/rig.rb"

# Make sure the Homebrew Tap is present
if [ ! -f "$HOMEBREW_FORMULA" ]; then
  echo "[ERROR] Could not find Rig Homebrew formula in: ${HOMEBREW_FORMULA}"
  exit 1
fi

# Package rig and any additional scripts
echo "[INFO] Packaging rig ${VERSION} for publishing"
mkdir -p $DIST_DIR
cp ${BASE_DIR}/scripts/docker-machine-watch-rsync.sh $BIN_DIR/.
cp ${BASE_DIR}/scripts/bash_autocomplete $BIN_DIR/.
cp ${BASE_DIR}/scripts/zsh_autocomplete $BIN_DIR/.
pushd $BIN_DIR
tar czf $DIST rig docker-machine-watch-rsync.sh bash_autocomplete zsh_autocomplete
popd

# Generate the checksum
SHA=`shasum -a 256 $DIST | awk '{print $1}'`
echo "[INFO] Rig ${VERSION} checksum is ${SHA}"

# Move the file to S3 for distribution
echo "[INFO] Deploying ${VERSION}"
aws s3 ls $S3_BUCKET | grep "${FILENAME}" && {
  read -p "[WARN] Version ${VERSION} is already published to S3. Overwrite? <y/N> " prompt
  if [[ ! $prompt =~ [yY](es)* ]]; then
    echo "Publish cancelled."
    exit 1
  fi
}

echo "[INFO] Writing to AWS/S3"
aws s3 cp $DIST s3://${S3_BUCKET}/${FILENAME} --acl public-read

# Write out the Homebrew formula
echo "[INFO] Writing Homebrew formula"
cat <<EOF > $HOMEBREW_FORMULA
class Rig < Formula
  desc "Containerized platform environment for projects. See https://outrigger.sh for documentation. "
  homepage "https://outrigger.sh"
  url "https://s3.amazonaws.com/${S3_BUCKET}/${FILENAME}"
  version "${VERSION}"
  sha256 "${SHA}"

  depends_on "docker"
  depends_on "docker-machine"
  depends_on "docker-compose"
  depends_on "docker-machine-nfs"

  def install
    bin.install "rig"
    bin.install "docker-machine-watch-rsync.sh"

    bash_completion.install "bash_autocomplete" => "rig"
    zsh_completion.install "zsh_autocomplete" => "_rig"
  end

  test do
    system "#{bin}/rig", "--version"
  end
end
EOF

echo "[NOTICE] Remember that you need to commit the formula change to homebrew-outrigger"
