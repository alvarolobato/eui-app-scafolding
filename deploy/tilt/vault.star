def vault_read(path, address='https://vault-ci-prod.elastic.dev'):
  result = local('vault read -format=json {}'.format(path), quiet=True, env={'VAULT_ADDR':address})
  return decode_json(result)
