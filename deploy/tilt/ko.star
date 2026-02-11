def ko_build(ref, path, platform='linux/amd64,linux/arm64', main_path='.', deps=[], **kwargs):
  commands = [
    "set -eo pipefail",
    "cd {}".format(path),
    "export KO_DOCKER_REPO={}".format('ko.local'),
    "export KOIMAGE=$(ko build --push=false --platform='{}' {})".format(platform, main_path),
    "docker tag $KOIMAGE $EXPECTED_REF",
  ]

  commands = commands + kwargs.get('commands', [])
  kwargs.pop("commands", "")

  custom_build(
    ref=ref,
    command=["bash", "-c", ";\n".join(commands)],
    deps=[path] + deps,
    **kwargs,
  )
  return ref
