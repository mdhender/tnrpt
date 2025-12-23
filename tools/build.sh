#!/bin/bash
# tools/build.sh - create a production distribution tarball in ./dist/prod

# Pre-flight checks
command -v gtar >/dev/null 2>&1 || {
  echo "❌ error: gtar not found. Install GNU tar."
  exit 2
}

[ -f "go.mod" ] || {
  echo "❌ error: must be run from root of the repository"
  exit 2
}

## fetch the version
VERSION=$( go run ./cmd/server -version )
[ "${VERSION}" == "" ] && {
  echo "❌ error: unable to fetch version information from server"
  exit 2
}
echo "📦  info: building version '${VERSION}'"
serverAssets="dist/prod/static"
serverBinary="dist/prod/bin/server"
serverConfig="dist/prod/config"
serverData="dist/prod/data"
tarballArtifact="tnrpt-${VERSION}.tgz"
prodTarball="dist/prod/${tarballArtifact}"

## remove and recreate the production deployment directory
echo "📦  info: clearing out dist/prod"
rm -rf dist/prod || {
  echo "❌ error: could not clear out dist/prod"
  exit 2
}
mkdir -p dist/prod/{bin,config,data,static} || {
  echo "❌ error: could not rebuild dist/prod"
  exit 2
}

## build the executable for linux
echo "🛠️  info: building '${serverBinary}'"
CGO_ENABLED=0    # make the executable as static as possible
GOOS=linux
GOARCH=amd64
GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} go build -o "${serverBinary}" ./cmd/server || {
  echo "❌ error: Go build failed"
  exit 2
}
echo "✅  info: created '${serverBinary}'"

## copy assets
echo "🛠️  info: building '${serverAssets}'"
cp -p web/static/* "${serverAssets}/" || {
  echo "❌ error: build assets failed"
  exit 2
}
echo "✅  info: created '${serverAssets}'"

## copy configuration
echo "🛠️  info: building '${serverConfig}'"
cp -p data/production/config/*.json "${serverConfig}/" || {
  echo "❌ error: build server configuration failed"
  exit 2
}
echo "✅  info: created '${serverConfig}'"

## build the deployment tarball
echo "🛠️  info: building '${prodTarball}'"
cd dist/prod || {
  echo "❌ error: failed to set def to dist/prod"
  exit 2
}
gtar -cz -f ${tarballArtifact} --exclude=".DS_Store" bin config data static || {
  echo "❌ error: failed to create tarball"
  exit 2
}
echo "✅  info: created tarball: ${prodTarball}"

exit 0
