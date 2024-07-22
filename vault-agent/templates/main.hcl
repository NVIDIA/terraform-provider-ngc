{{- with secret "kv/gitlab/common/artifactory/sw-backstage-terraform/deploy-key" }}
ARTIFACTORY_PRODUCTION_SECRET={{ .Data.data.access_token }}
{{- end }}

export ARTIFACTORY_PRODUCTION_SECRET
