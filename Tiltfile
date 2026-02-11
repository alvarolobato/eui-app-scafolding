# vim: set ft=starlark:
load('./deploy/tilt/ko.star', 'ko_build')
load('./deploy/tilt/vault.star', 'vault_read')
load('ext://secret', 'secret_from_dict')
load('ext://helm_resource', 'helm_resource', 'helm_repo')

ko_build('app-backend', 'backend')
ko_build('app-proxy', './deploy/dev/app-proxy')
docker_build('app-frontend', 'frontend', dockerfile="frontend/dev/Dockerfile.tilt",
  live_update=[
    fall_back_on(['frontend/package.json', 'frontend/yarn.lock']),
    sync('frontend/src', '/app/src'),
  ]
)

# Deploy Elasticsearch and Kibana with ECK.
helm_repo('elastic', 'https://helm.elastic.co/', labels='stack')
helm_resource('eck-operator', 'elastic/eck-operator', labels='stack')
k8s_yaml(os.path.join('deploy', 'dev', 'elasticsearch.yaml'))
k8s_resource(
    objects=['elasticsearch:Elasticsearch:default'], new_name='elasticsearch', port_forwards=9200, labels='stack',
    pod_readiness='wait',
    extra_pod_selectors=[{'common.k8s.elastic.co/type': 'elasticsearch'}],
    discovery_strategy='selectors-only',
    resource_deps=['eck-operator'],
)
k8s_resource(objects=['elasticsearch-admin:Secret:default'], new_name='elasticsearch-credentials', labels='stack')
k8s_yaml(os.path.join('deploy', 'dev', 'kibana.yaml'))
k8s_resource(
    objects=['kibana:Kibana:default'], new_name='kibana', port_forwards=5601, labels='stack',
    pod_readiness='wait',
    extra_pod_selectors=[{'common.k8s.elastic.co/type': 'kibana'}],
    discovery_strategy='selectors-only',
    resource_deps=['eck-operator', 'elasticsearch'],
)
k8s_yaml(os.path.join('deploy', 'dev', 'apm-server.yaml'))
k8s_resource(
    objects=['apm-server:ApmServer:default'], new_name='apm-server', labels='stack',
    pod_readiness='wait',
    extra_pod_selectors=[{'common.k8s.elastic.co/type': 'apm-server'}],
    discovery_strategy='selectors-only',
    resource_deps=['eck-operator', 'elasticsearch'],
)

# When Elasticsearch is ready, run a script to create the indices and an API Key secret.
local_resource('init-elasticsearch', cmd='deploy/dev/init_es.sh', resource_deps=['elasticsearch'], labels='stack')

# Create secrets from local files or Vault.
def create_secrets(name):
    yaml_path = os.path.join('deploy/dev/secrets/{}.yaml'.format(name))
    vault_path = 'secret/ci/elastic-app/dev/{}'.format(name)
    inputs = {}
    if os.path.exists(yaml_path):
        print('Creating secret "{}" from local file: {}'.format(name, yaml_path))
        inputs = decode_yaml(read_file(yaml_path))
    else:
        print('WARNING: Secret file not found: {}. Please create it or configure Vault.'.format(yaml_path))
        print('  Expected format for {}: see deploy/dev/secrets/{}.yaml.example'.format(name, name))
        # Create empty secret to allow Tilt to start (will fail at runtime)
        inputs = {}
    k8s_yaml(secret_from_dict(name, inputs=inputs))

create_secrets('google')
k8s_yaml(secret_from_dict('app', inputs={'admin_secret': 'hunter2', 'encryption_keys': ''}))

# Deploy the Helm chart.
k8s_yaml(helm(
  os.path.join('deploy', 'helm'),
  set=[
    'backend.image=app-backend',
    'frontend.image=app-frontend',
    'ingress.enabled=false',
    'imagePullSecret=',

    # Don't create Vault secret claims
    'secret.vault.enabled=false',

    'elasticsearch.url=http://elasticsearch-es-http:9200',
    'observability.server_url=http://apm-server-apm-http:8200',
    'observability.use_secret_token=false',
  ]
))

# Deploy app-proxy for local development. This serves TLS, reverse-proxying
# "/api" to app-backend and everything else to app-frontend.
k8s_yaml(helm('./deploy/dev/app-proxy/helm'))
k8s_resource('app-proxy', port_forwards='8443')
k8s_resource('app-backend', port_forwards='4000')
