{{/*
Expand the name of the chart.
*/}}
{{- define "blockasaurus.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name.
*/}}
{{- define "blockasaurus.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label.
*/}}
{{- define "blockasaurus.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "blockasaurus.labels" -}}
helm.sh/chart: {{ include "blockasaurus.chart" . }}
{{ include "blockasaurus.selectorLabels" . }}
app.kubernetes.io/version: {{ .Values.image.tag | default .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "blockasaurus.selectorLabels" -}}
app.kubernetes.io/name: {{ include "blockasaurus.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name.
*/}}
{{- define "blockasaurus.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "blockasaurus.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Validate the query log volume against the query log target.

Mounting a volume the query log does not write to is a silent failure: the
database lands wherever the target points -- the config PVC, or the container's
read-only root -- and the mounted volume sits there empty. Catch it at render
time instead, where the message can say what to change.
*/}}
{{- define "blockasaurus.validateQueryLogVolume" -}}
{{- if .Values.queryLogVolume.enabled }}
{{- $mountPath := required "queryLogVolume.mountPath is required when queryLogVolume.enabled" .Values.queryLogVolume.mountPath }}
{{- if not (has .Values.queryLogVolume.type (list "hostPath" "emptyDir")) }}
{{- fail (printf "queryLogVolume.type must be hostPath or emptyDir, got %q" .Values.queryLogVolume.type) }}
{{- end }}
{{- $target := (.Values.config.queryLog).target | default "" }}
{{- if eq ((.Values.config.queryLog).type | default "") "sqlite" }}
{{- if not (hasPrefix (printf "%s/" $mountPath) $target) }}
{{- fail (printf "config.queryLog.target (%q) must be a file under queryLogVolume.mountPath (%q), or set queryLogVolume.enabled=false to keep the query log on the config volume" $target $mountPath) }}
{{- end }}
{{- end }}
{{- if eq .Values.queryLogVolume.type "hostPath" }}
{{- if not (.Values.queryLogVolume.hostPath).path }}
{{- fail "queryLogVolume.hostPath.path is required when queryLogVolume.type is hostPath" }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
uid:gid the query log directory is handed to.

Mirrors the pod's own identity so the init container's chown matches whoever
actually opens the database. Defaults to 100:100, the USER the image declares.
*/}}
{{- define "blockasaurus.queryLogOwner" -}}
{{- $uid := (.Values.securityContext).runAsUser | default 100 }}
{{- $gid := (.Values.podSecurityContext).fsGroup | default ((.Values.securityContext).runAsGroup | default 100) }}
{{- printf "%v:%v" $uid $gid }}
{{- end }}
