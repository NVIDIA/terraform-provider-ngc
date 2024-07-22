{{- with secret "kv/gitlab/common/ssa/yoidq40xarydiw2q17k9u5wi0aimk22yb3dxc-tpyz8/clients/pulse-scan" }}
SF_CLIENT_ID={{ .Data.data.id }}
SF_CLIENT_SECRET={{ .Data.data.secret }}
{{- end }}

export SF_CLIENT_ID
export SF_CLIENT_SECRET
